---
title: Interpolation
---

Benthos allows you to dynamically set config fields with environment variables anywhere within a config file using the syntax `${<variable-name>}` (or `${<variable-name>:<default-value>}` in order to specify a default value). This is useful for setting environment specific fields such as addresses:

```yaml
input:
  kafka_balanced:
    addresses: [ "${BROKERS}" ]
    consumer_group: benthos_bridge_consumer
    topics: [ "haha_business" ]
```

```sh
BROKERS="foo:9092,bar:9092" benthos -c ./config.yaml
```

If a literal string is required that matches this pattern (`${foo}`) you can escape it with double brackets. For example, the string `${{foo}}` is read as the literal `${foo}`.

## Bloblang Queries

Some Benthos fields also support Bloblang function interpolations, which are much more powerful expressions that allow you to query the contents of messages and perform arithmetic. The syntax of a function interpolation is `${!<bloblang expression>}`, where the contents are a logical expression including a range of functions. For example, with the following config:

```yaml
output:
  kafka:
    addresses: [ "TODO:6379" ]
    topic: 'dope-${!json("topic")}'
```

A message with the contents `{"topic":"foo","message":"hello world"}` would be routed to the Kafka topic `dope-foo`.

If a literal string is required that matches this pattern (`${!foo}`) then, similar to environment variables, you can escape it with double brackets. For example, the string `${{!foo}}` would be read as the literal `${!foo}`.

Bloblang supports arithmetic, boolean operators, coalesce and mapping expressions, there are some [examples of this below](#examples).

## Expressions

### Pattern Matching

A `match` expression allows you to perform pattern matching on values:

```
match json().type
  this == "article" => json().article
  this == "comment" => json().comment
  _ => json()
```

## Functions

### `content()`

Returns the full contents of a message.

### `deleted()`

This is a special function indicating that the mapping target should be deleted. For example, it can be used to remove elements of an array within `for_each`:

```
json("foo").for_each(
  match this
    this < 10 => deleted()
    _ => this - 10
)
```

Given the input message `{"foo":[11,12,7,13]}` the above mapping would result in `[1,2,3]`.

### `error()`

If an error has occurred during the processing of a message this function returns the reported cause of the error. For more information about error
handling patterns read [here][error_handling].

### `batch_size()`

Returns the size of the message batch.

### `json(string)`

Returns the value of a field within a JSON message located by a [dot path][field_paths] argument. For example, with a message `{"foo":{"bar":"hello world"}}` the function `${!json("foo.bar")}` would return `hello world`.

The parameter is optional and if omitted the entire JSON payload is returned.

### `meta(string)`

Returns the value of a metadata key from a message identified by a key. Message metadata can be modified using the [metadata processor][meta_proc].

The parameter is optional and if omitted the entire metadata contents are returned as a JSON object.

### `uuid_v4()`

Generates a new RFC-4122 UUID each time it is invoked and prints a string representation.

### `timestamp_unix_nano()`

Resolves to the current unix timestamp in nanoseconds. E.g. `foo ${!timestamp_unix_nano()} bar` prints `foo 1517412152475689615 bar`.

### `timestamp_unix(int)`

Resolves to the current unix timestamp in seconds. E.g. `foo ${!timestamp_unix()} bar` prints `foo 1517412152 bar`. You can add fractional precision up to the nanosecond by specifying the precision as an argument, e.g. `${!timestamp_unix(3)}` for millisecond precision.

### `timestamp(string)`

Prints the current time in a custom format specified by the argument. The format is defined by showing how the reference time, defined to be
`Mon Jan 2 15:04:05 -0700 MST 2006` would be displayed if it were the value.

A fractional second is represented by adding a period and zeros to the end of the seconds section of layout string, as in `15:04:05.000` to format a time stamp with millisecond precision.

### `timestamp_utc(string)`

The equivalent of `timestamp` except the time is printed as UTC instead of the local timezone.

### `count(string)`

The `count` function is a counter starting at 1 which increments after each time it is called. Count takes an argument which is an identifier for the counter, allowing you to specify multiple unique counters in your configuration.

### `hostname()`

Resolves to the hostname of the machine running Benthos. E.g. `foo ${!hostname()} bar` might resolve to `foo glados bar`.

## Methods

Methods mutate the behaviour or result of a function (or literal).

### `catch(function)`

If the result of the target function fails (due to incorrect types, failed parsing, etc) the argument is returned instead.

### `for_each(function)`

Apply a function to each element of an array and replace the element with the result.

### `from(int)`

Execute a function from the context of another message in the batch. This allows you to mutate events based on the contents of other messages.

For example, `json("foo").from(1)` would extract the contents of the JSON field `foo` specifically from message index `1` of a batch.

### `from_all()`

Execute a function for all messages of the batch, and return an array of all results.

For example, `json("foo").from_all()` would extract the contents of the JSON field `foo` from all messages and return an array of the values.

### `map(function)`

Apply a function to the result of another function, allowing you to perform mappings on the result.

### `or(function)`

If the result of the target function fails or resolves to `null`, returns the argument instead.

### `sum()`

Sum the numerical values of an array. E.g. `json("foo").from_all().sum()` extracts the value of `foo` from all messages of a batch and adds them.

## Examples

### Coalesce and Mapping

Bloblang supports coalesce and mapping, which makes it easy to extract values from slightly varying data structures:

```yaml
pipeline:
  processors:
  - json:
      operator: set
      path: result
      value: '${! json().foo.(a | b | c).baz }'
```

Here's a map of inputs to resulting values:

```
{"foo":{"a":{"baz":"from_a"},"c":{"baz":"from_c"}}} -> from_a
{"foo":{"b":{"baz":"from_b"},"c":{"baz":"from_c"}}} -> from_b
{"foo":{"b":null,"c":{"baz":"from_c"}}}             -> from_c
```

### Delayed Processing

We have a stream of JSON documents each with a unix timestamp field `doc.received_at` which is set when our platform receives it. We wish to only process messages an hour _after_ they were received. We can achieve this by running the `sleep` processor using an interpolation function to calculate the seconds needed to wait for:

```yaml
pipeline:
  processors:
  - sleep:
      duration: '${! 3600 - ( timestamp_unix() - json("doc.created_at") ) }s'
```

If the calculated result is less than or equal to zero the processor does not sleep at all. If the value of `doc.created_at` is a string Benthos will make a best attempt to parse it as a number.

[env_var_config]: https://github.com/Jeffail/benthos/blob/master/config/env/default.yaml
[error_handling]: /docs/configuration/error_handling
[field_paths]: /docs/configuration/field_paths
[meta_proc]: /docs/components/processors/metadata