package processor

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/response"
	"github.com/Jeffail/benthos/v3/lib/types"
	"github.com/Jeffail/benthos/v3/lib/x/docs"
	"github.com/opentracing/opentracing-go"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeDecode] = TypeSpec{
		constructor: NewDecode,
		Summary: `
Decodes messages according to the selected scheme.`,
		FieldSpecs: docs.FieldSpecs{
			docs.FieldCommon("scheme", "The decoding scheme to use.").HasOptions("hex", "base64"),
			partsFieldSpec,
		},
	}
}

//------------------------------------------------------------------------------

// DecodeConfig contains configuration fields for the Decode processor.
type DecodeConfig struct {
	Scheme string `json:"scheme" yaml:"scheme"`
	Parts  []int  `json:"parts" yaml:"parts"`
}

// NewDecodeConfig returns a DecodeConfig with default values.
func NewDecodeConfig() DecodeConfig {
	return DecodeConfig{
		Scheme: "base64",
		Parts:  []int{},
	}
}

//------------------------------------------------------------------------------

type decodeFunc func(bytes []byte) ([]byte, error)

func base64Decode(b []byte) ([]byte, error) {
	e := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(b))
	return ioutil.ReadAll(e)
}

func hexDecode(b []byte) ([]byte, error) {
	e := hex.NewDecoder(bytes.NewReader(b))
	return ioutil.ReadAll(e)
}

func strToDecoder(str string) (decodeFunc, error) {
	switch str {
	case "base64":
		return base64Decode, nil
	case "hex":
		return hexDecode, nil
	}
	return nil, fmt.Errorf("decode scheme not recognised: %v", str)
}

//------------------------------------------------------------------------------

// Decode is a processor that can selectively decode parts of a message
// following a chosen scheme.
type Decode struct {
	conf DecodeConfig
	fn   decodeFunc

	log   log.Modular
	stats metrics.Type

	mCount     metrics.StatCounter
	mErr       metrics.StatCounter
	mSent      metrics.StatCounter
	mBatchSent metrics.StatCounter
}

// NewDecode returns a Decode processor.
func NewDecode(
	conf Config, mgr types.Manager, log log.Modular, stats metrics.Type,
) (Type, error) {
	cor, err := strToDecoder(conf.Decode.Scheme)
	if err != nil {
		return nil, err
	}
	return &Decode{
		conf:  conf.Decode,
		fn:    cor,
		log:   log,
		stats: stats,

		mCount:     stats.GetCounter("count"),
		mErr:       stats.GetCounter("error"),
		mSent:      stats.GetCounter("sent"),
		mBatchSent: stats.GetCounter("batch.sent"),
	}, nil
}

//------------------------------------------------------------------------------

// ProcessMessage applies the processor to a message, either creating >0
// resulting messages or a response to be sent back to the message source.
func (c *Decode) ProcessMessage(msg types.Message) ([]types.Message, types.Response) {
	c.mCount.Incr(1)
	newMsg := msg.Copy()

	proc := func(i int, span opentracing.Span, part types.Part) error {
		newBytes, err := c.fn(part.Get())
		if err != nil {
			c.log.Errorf("Failed to decode message part: %v\n", err)
			c.mErr.Incr(1)
			return err
		}
		part.Set(newBytes)
		return nil
	}

	if newMsg.Len() == 0 {
		return nil, response.NewAck()
	}

	IteratePartsWithSpan(TypeDecode, c.conf.Parts, newMsg, proc)

	c.mBatchSent.Incr(1)
	c.mSent.Incr(int64(newMsg.Len()))
	msgs := [1]types.Message{newMsg}
	return msgs[:], nil
}

// CloseAsync shuts down the processor and stops processing requests.
func (c *Decode) CloseAsync() {
}

// WaitForClose blocks until the processor has closed down.
func (c *Decode) WaitForClose(timeout time.Duration) error {
	return nil
}

//------------------------------------------------------------------------------
