---
title: http
type: processor
---

<!--
     THIS FILE IS AUTOGENERATED!

     To make changes please edit the contents of:
     lib/processor/http.go
-->


Performs an HTTP request using a message batch as the request body, and replaces
the original message parts with the body of the response.


import Tabs from '@theme/Tabs';

<Tabs defaultValue="common" values={[
  { label: 'Common', value: 'common', },
  { label: 'Advanced', value: 'advanced', },
]}>

import TabItem from '@theme/TabItem';

<TabItem value="common">

```yaml
# Common config fields, showing default values
http:
  parallel: false
  max_parallel: 0
  request:
    url: http://localhost:4195/post
    verb: POST
    headers:
      Content-Type: application/octet-stream
    rate_limit: ""
    timeout: 5s
```

</TabItem>
<TabItem value="advanced">

```yaml
# All config fields, showing default values
http:
  parallel: false
  max_parallel: 0
  request:
    url: http://localhost:4195/post
    verb: POST
    headers:
      Content-Type: application/octet-stream
    oauth:
      access_token: ""
      access_token_secret: ""
      consumer_key: ""
      consumer_secret: ""
      enabled: false
      request_url: ""
    basic_auth:
      enabled: false
      password: ""
      username: ""
    tls:
      enabled: false
      skip_cert_verify: false
      root_cas_file: ""
      client_certs: []
    copy_response_headers: false
    rate_limit: ""
    timeout: 5s
    retry_period: 1s
    max_retry_backoff: 300s
    retries: 3
    backoff_on:
    - 429
    drop_on: []
    successful_on: []
```

</TabItem>
</Tabs>

If the batch contains only a single message part then it will be sent as the
body of the request. If the batch contains multiple messages then they will be
sent as a multipart HTTP request using a `Content-Type: multipart`
header.

If you are sending batches and wish to avoid this behaviour then you can set the
`parallel` flag to `true` and the messages of a batch will
be sent as individual requests in parallel. You can also cap the max number of
parallel requests with `max_parallel`. Alternatively, you can use the
[`archive`](/docs/components/processors/archive) processor to create a single message
from the batch.

The `rate_limit` field can be used to specify a rate limit
[resource](/docs/components/rate_limits/about) to cap the rate of requests across all
parallel components service wide.

The URL and header values of this type can be dynamically set using function
interpolations described [here](/docs/configuration/interpolation#functions).

In order to map or encode the payload to a specific request body, and map the
response back into the original payload instead of replacing it entirely, you
can use the [`process_map`](/docs/components/processors/process_map) or
 [`process_field`](/docs/components/processors/process_field) processors.

## Response Codes

Benthos considers any response code between 200 and 299 inclusive to indicate a
successful response, you can add more success status codes with the field
`successful_on`.

When a request returns a response code within the `backoff_on` field
it will be retried after increasing intervals.

When a request returns a response code within the `drop_on` field it
will not be reattempted and is immediately considered a failed request.

## Adding Metadata

If the request returns a response code this processor sets a metadata field
`http_status_code` on all resulting messages.

If the field `copy_response_headers` is set to `true` then any headers
in the response will also be set in the resulting message as metadata.
 
## Error Handling

When all retry attempts for a message are exhausted the processor cancels the
attempt. These failed messages will continue through the pipeline unchanged, but
can be dropped or placed in a dead letter queue according to your config, you
can read about these patterns [here](/docs/configuration/error_handling).

## Fields

### `parallel`

When processing batched messages, whether to send messages of the batch in parallel, otherwise they are sent within a single request.


Type: `bool`  
Default: `false`  

### `max_parallel`

A limit on the maximum messages in flight when sending batched messages in parallel.


Type: `number`  
Default: `0`  

### `request`

Controls how the HTTP request is made.


Type: `object`  
Default: `{"backoff_on":[429],"basic_auth":{"enabled":false,"password":"","username":""},"copy_response_headers":false,"drop_on":[],"headers":{"Content-Type":"application/octet-stream"},"max_retry_backoff":"300s","oauth":{"access_token":"","access_token_secret":"","consumer_key":"","consumer_secret":"","enabled":false,"request_url":""},"rate_limit":"","retries":3,"retry_period":"1s","successful_on":[],"timeout":"5s","tls":{"client_certs":[],"enabled":false,"root_cas_file":"","skip_cert_verify":false},"url":"http://localhost:4195/post","verb":"POST"}`  

### `request.url`

The URL to connect to.
This field supports [interpolation functions](/docs/configuration/interpolation#functions).


Type: `string`  
Default: `"http://localhost:4195/post"`  

### `request.verb`

A verb to connect with


Type: `string`  
Default: `"POST"`  

```yaml
# Examples

verb: POST

verb: GET

verb: DELETE
```

### `request.headers`

A map of headers to add to the request.
This field supports [interpolation functions](/docs/configuration/interpolation#functions).


Type: `object`  
Default: `{"Content-Type":"application/octet-stream"}`  

```yaml
# Examples

headers:
  Content-Type: application/octet-stream
```

### `request.oauth`

Allows you to specify open authentication.


Type: `object`  
Default: `{"access_token":"","access_token_secret":"","consumer_key":"","consumer_secret":"","enabled":false,"request_url":""}`  

```yaml
# Examples

oauth:
  access_token: baz
  access_token_secret: bev
  consumer_key: foo
  consumer_secret: bar
  enabled: true
  request_url: http://thisisjustanexample.com/dontactuallyusethis
```

### `request.basic_auth`

Allows you to specify basic authentication.


Type: `object`  
Default: `{"enabled":false,"password":"","username":""}`  

```yaml
# Examples

basic_auth:
  enabled: true
  password: bar
  username: foo
```

### `request.tls`

Custom TLS settings can be used to override system defaults.


Type: `object`  
Default: `{"client_certs":[],"enabled":false,"root_cas_file":"","skip_cert_verify":false}`  

### `request.tls.enabled`

Whether custom TLS settings are enabled.


Type: `bool`  
Default: `false`  

### `request.tls.skip_cert_verify`

Whether to skip server side certificate verification.


Type: `bool`  
Default: `false`  

### `request.tls.root_cas_file`

The path of a root certificate authority file to use.


Type: `string`  
Default: `""`  

### `request.tls.client_certs`

A list of client certificates to use.


Type: `array`  
Default: `[]`  

```yaml
# Examples

client_certs:
- cert: foo
  key: bar

client_certs:
- cert_file: ./example.pem
  key_file: ./example.key
```

### `request.copy_response_headers`

Sets whether to copy the headers from the response to the resulting payload.


Type: `bool`  
Default: `false`  

### `request.rate_limit`

An optional [rate limit](/docs/components/rate_limits/about) to throttle requests by.


Type: `string`  
Default: `""`  

### `request.timeout`

A static timeout to apply to requests.


Type: `string`  
Default: `"5s"`  

### `request.retry_period`

The base period to wait between failed requests.


Type: `string`  
Default: `"1s"`  

### `request.max_retry_backoff`

The maximum period to wait between failed requests.


Type: `string`  
Default: `"300s"`  

### `request.retries`

The maximum number of retry attempts to make.


Type: `number`  
Default: `3`  

### `request.backoff_on`

A list of status codes whereby retries should be attempted but the period between them should be increased gradually.


Type: `array`  
Default: `[429]`  

### `request.drop_on`

A list of status codes whereby the attempt should be considered failed but retries should not be attempted.


Type: `array`  
Default: `[]`  

### `request.successful_on`

A list of status codes whereby the attempt should be considered successful (allows you to configure non-2XX codes).


Type: `array`  
Default: `[]`  


