package query

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/Jeffail/benthos/v3/lib/bloblang/x/parser"
	"golang.org/x/xerrors"
)

//------------------------------------------------------------------------------

type badFunctionErr string

func (e badFunctionErr) Error() string {
	/*
		exp := []string{}
		for k := range functions {
			exp = append(exp, k)
		}
		sort.Strings(exp)
		return fmt.Sprintf("unrecognised function '%v', expected one of: %v", string(e), exp)
	*/
	return fmt.Sprintf("unrecognised function '%v'", string(e))
}

func (e badFunctionErr) ToExpectedErr() parser.ExpectedError {
	exp := []string{}
	for k := range functions {
		exp = append(exp, k)
	}
	sort.Strings(exp)
	return parser.ExpectedError(exp)
}

type badMethodErr string

func (e badMethodErr) Error() string {
	/*
		exp := []string{}
		for k := range methods {
			exp = append(exp, k)
		}
		sort.Strings(exp)
		return fmt.Sprintf("unrecognised method '%v', expected one of: %v", string(e), exp)
	*/
	return fmt.Sprintf("unrecognised method '%v'", string(e))
}

func (e badMethodErr) ToExpectedErr() parser.ExpectedError {
	exp := []string{}
	for k := range methods {
		exp = append(exp, k)
	}
	sort.Strings(exp)
	return parser.ExpectedError(exp)
}

//------------------------------------------------------------------------------

func functionArgsParser(allowFunctions bool) parser.Type {
	paramTypes := []parser.Type{
		parser.Boolean(),
		parser.Number(),
		parser.QuotedString(),
	}
	parseStart := parser.Char('(')
	parseEnd := parser.Char(')')
	parseNext := parser.AnyOf(
		parser.Char(')'),
		parser.Char(','),
	)
	whitespace := parser.DiscardAll(
		parser.AnyOf(
			parser.Char('\n'),
			parser.SpacesAndTabs(),
		),
	)

	return func(input []rune) parser.Result {
		if allowFunctions {
			paramTypes = append(paramTypes, createParser(false))
		}
		parseParam := parser.AnyOf(paramTypes...)

		res := parseStart(input)
		if res.Err != nil {
			return res
		}

		var params []interface{}
		if earlyExit := parseEnd(res.Remaining); earlyExit.Err == nil {
			earlyExit.Result = params
			return earlyExit
		}

		for {
			res = whitespace(res.Remaining)
			i := len(input) - len(res.Remaining)
			res = parseParam(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}
			params = append(params, res.Result)

			res = whitespace(res.Remaining)
			i = len(input) - len(res.Remaining)
			res = parseNext(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}

			if ")" == res.Result.(string) {
				break
			}
		}

		return parser.Result{
			Result:    params,
			Err:       nil,
			Remaining: res.Remaining,
		}
	}
}

func nullParser() parser.Type {
	nullMatch := parser.Match("null")
	return func(input []rune) parser.Result {
		res := nullMatch(input)
		if res.Err == nil {
			res.Result = nil
		}
		return res
	}
}

func literalParser() parser.Type {
	parseLiteral := parser.AnyOf(
		parser.Boolean(),
		parser.Number(),
		parser.QuotedString(),
		nullParser(),
	)
	return func(input []rune) parser.Result {
		res := parseLiteral(input)
		if res.Err == nil {
			res.Result = literalFunction(res.Result)
		}
		return res
	}
}

func fieldLiteralParser(ctxFn Function, allowRoot, supportThis bool) parser.Type {
	thisParser := parser.Match("this")
	fieldPathParser := parser.AnyOf(
		parser.InRange('a', 'z'),
		parser.InRange('A', 'Z'),
		parser.InRange('0', '9'),
		parser.InRange('*', '-'),
		parser.Char('_'),
		parser.Char('~'),
	)
	return func(input []rune) parser.Result {
		res := parser.Result{
			Remaining: input,
		}

		var fn Function
		var err error

		if supportThis {
			res = thisParser(res.Remaining)
			if res.Err == nil {
				if res = parser.Char('.')(res.Remaining); res.Err != nil {
					fn, err = fieldFunction()
				}
			} else if !allowRoot {
				return res
			}
		}
		if fn == nil && err == nil {
			var buf bytes.Buffer
			for {
				if res = fieldPathParser(res.Remaining); res.Err != nil {
					break
				}
				buf.WriteString(res.Result.(string))
			}
			if buf.Len() == 0 {
				if res.Err == nil {
					res.Err = parser.ExpectedError{"field-path"}
				}
				return parser.Result{
					Remaining: input,
					Err:       res.Err,
				}
			}
			fn, err = fieldFunction(buf.String())
		}

		if err == nil && ctxFn != nil {
			fn, err = mapMethod(ctxFn, fn)
		}
		if err != nil {
			return parser.Result{
				Remaining: input,
				Err:       err,
			}
		}

		for {
			res = parser.Char('.')(res.Remaining)
			if res.Err != nil {
				break
			}

			i := len(input) - len(res.Remaining)
			res = parseFunctionTail(fn)(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}
			fn = res.Result.(Function)
		}

		return parser.Result{
			Remaining: res.Remaining,
			Result:    fn,
		}
	}
}

func parseFunctionTail(fn Function) parser.Type {
	openBracket := parser.Char('(')
	closeBracket := parser.Char(')')

	tailRootParser := parser.AnyOf(
		openBracket,
		parseMethod(fn),
		fieldLiteralParser(fn, true, false),
	)

	return func(input []rune) parser.Result {
		res := tailRootParser(input)
		if res.Err != nil {
			return res
		}
		if _, isStr := res.Result.(string); isStr {
			res = parser.SpacesAndTabs()(res.Remaining)
			i := len(input) - len(res.Remaining)
			res = Parse(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}
			mapFn := res.Result.(Function)
			res = parser.SpacesAndTabs()(res.Remaining)
			i = len(input) - len(res.Remaining)
			res = closeBracket(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}
			if res.Result, res.Err = mapMethod(fn, mapFn); res.Err != nil {
				res.Remaining = input
			}
			return res
		}
		return res
	}
}

func parseMethod(fn Function) parser.Type {
	argsParser := functionArgsParser(true)

	return func(input []rune) parser.Result {
		res := parser.SnakeCase()(input)
		if res.Err != nil {
			return res
		}

		targetMethod := res.Result.(string)
		if len(res.Remaining) == 0 {
			return parser.Result{
				Err: parser.ErrAtPosition(
					len(input),
					parser.ExpectedError{"method-parameters"},
				),
				Remaining: input,
			}
		}

		i := len(input) - len(res.Remaining)
		res = argsParser(res.Remaining)
		if res.Err != nil {
			return parser.Result{
				Err: parser.ErrAtPosition(i, res.Err).Expand(
					func(err error) error {
						return xerrors.Errorf("failed to parse method arguments: %w", err)
					},
				),
				Remaining: input,
			}
		}
		args := res.Result.([]interface{})

		mtor, exists := methods[targetMethod]
		if !exists {
			return parser.Result{
				Err:       badMethodErr(targetMethod),
				Remaining: input,
			}
		}

		method, err := mtor(fn, args...)
		if err != nil {
			return parser.Result{
				Err:       parser.ErrAtPosition(i, err),
				Remaining: input,
			}
		}
		return parser.Result{
			Result:    method,
			Remaining: res.Remaining,
		}
	}
}

func functionParser() parser.Type {
	argsParser := functionArgsParser(true)

	return func(input []rune) parser.Result {
		var targetFunc string
		var args []interface{}

		res := parser.SnakeCase()(input)
		if res.Err != nil {
			return res
		}
		targetFunc = res.Result.(string)
		if len(res.Remaining) == 0 {
			return parser.Result{
				Err: parser.ErrAtPosition(
					len(input),
					parser.ExpectedError{"function-parameters"},
				),
				Remaining: input,
			}
		}

		i := len(input) - len(res.Remaining)
		res = argsParser(res.Remaining)
		if res.Err != nil {
			return parser.Result{
				Err: parser.ErrAtPosition(i, res.Err).Expand(
					func(err error) error {
						return xerrors.Errorf("failed to parse function arguments: %w", err)
					},
				),
				Remaining: input,
			}
		}
		args = res.Result.([]interface{})

		ftor, exists := functions[targetFunc]
		if !exists {
			return parser.Result{
				Err:       badFunctionErr(targetFunc),
				Remaining: input,
			}
		}

		fnResolver, err := ftor(args...)
		if err != nil {
			return parser.Result{
				Err:       parser.ErrAtPosition(i, err),
				Remaining: input,
			}
		}

		for {
			res = parser.Char('.')(res.Remaining)
			if res.Err != nil {
				break
			}

			i = len(input) - len(res.Remaining)
			res = parseFunctionTail(fnResolver)(res.Remaining)
			if res.Err != nil {
				res.Err = parser.ErrAtPosition(i, res.Err)
				res.Remaining = input
				return res
			}
			fnResolver = res.Result.(Function)
		}

		return parser.Result{
			Result:    fnResolver,
			Remaining: res.Remaining,
		}
	}
}

//------------------------------------------------------------------------------

func parseDeprecatedFunction(input []rune) parser.Result {
	var targetFunc, arg string

	for i := 0; i < len(input); i++ {
		if input[i] == ':' {
			targetFunc = string(input[:i])
			arg = string(input[i+1:])
		}
	}
	if len(targetFunc) == 0 {
		targetFunc = string(input)
	}

	ftor, exists := deprecatedFunctions[targetFunc]
	if !exists {
		return parser.Result{
			// Make no suggestions, we want users to move off of these functions
			Err:       parser.ExpectedError{},
			Remaining: input,
		}
	}
	return parser.Result{
		Result:    wrapDeprecatedFunction(ftor(arg)),
		Err:       nil,
		Remaining: nil,
	}
}

//------------------------------------------------------------------------------
