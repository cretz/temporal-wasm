# Spec

* Only memory is module memory, no importable host memory

## Exported Functions

### `run` function

Each WASM workflow will have a single exported `run` function.

* No params
* No return

## Host Types

### `bytes` type

Base64'd JSON string

### `failure` type

Fields:

* `message?: string`
* `type?: string`
* `non_retryable?: bool`
* `details?: []payload`
* `cause?: failure`

### `info` type

Fields:

* `params?: []payload`

### `log_level` type

Enum of `u32`, values:

* `Error` - `1`
* `Warn` - `2`
* `Info` - `3`
* `Debug` - `4`
* `Trace` - `5`

### `payload` type

Fields:

* `metadata?: map<string, bytes>`
* `data: bytes`

## Host Functions

### `complete` function

Complete a workflow immediately when reached.

* Params
  * `output_offset: u32` - JSON form of `[]payload`, immutable, static lifetime
  * `output_len: u32` - Can be 0
* No return

### `complete_with_failure` function

Complete a workflow immediately when reached with a failure.

* Params
  * `failure_offset: u32` - JSON form of `failure`, immutable, static lifetime
  * `failure_len: u32` - Can be 0
* No return

### `get_info` function

Get the immutable workflow info.

* Params
  * `info_offset: u32` - Where to write the `info` JSON
  * `info_len: u32` - Only used for validation, comes from `get_info_len`
* No return

### `get_info_len` function

Get the length of the info bytes for use in `get_info`.

* No params
* Return
  * `info_len: u32` - Byte length of info

### `write_log` function

Write a log

* Params
  * `level: log_level` - Log level (invalid value means no log)
  * `message_offset: u32` - UTF-8 string offset
  * `message_len: u32` - Byte length of the string
* No return

## Q/A

**Why not use Protobuf in/out?**

JSON is more ubiquitous and most languages that compile to WASM have JSON encoding/decoding. Not only do Protobuf
libraries _inside_ the WASM bundle add overhead, many compile-to-WASM languages don't have Protobuf implementations. And
it's not really needed here because we're not overly concerned about IO performance or size between WASM and host.

**Why is there not an "all_waiting" type of call to notify host that all coroutines are yielding?**

This becomes very language specific and in some languages as part of WASM compile, you can't tell when all coroutines
are yielding.

This means workflow commands will be sent as soon as they are triggered instead of batched once all coroutines are
suspended. More discussion on this is warranted.

**Why use high-level workflow calls instead of low-level events/commands?**

Since WASM is a mostly opaque runtime, there is no need to ask languages to have a large runtime. For example, Go WASM
can use all the native Go concurrency constructs and only concern itself with some of the activity and child workflow
invocations and not the details of low-level command/event proto building.

Also, by not asking WASM bundles to implement all of the middle-layer between low-level and high-level, it is easier to
build the harnesses to compile to WASM bundles. Shifting the burden to hosts which already have this implemented allows
us to encourage more WASM languages at a cost of fewer host languages.

**How do you implement activities in WASM?**

You don't. Currently we don't require the host implementations to even support local activities or side effects.

Eventually there may need to be local activity support which executes functions inside the host for performance reasons.
But for now, having the host only worry about workflows means any host can run a WASM workflow and be independent of its
activities.

Also, making workflows standalone bundles also allows ease of bundling for deployment/upload/versioning purposes in
different environments. It also encourages workflow and activity code to be developed independently and even run on
workers independently (which adds value as their scaling profiles are often not the same).

As part of WASM workflows, there is a plan to make the concept of "HTTP activities" standard. This essentially means any
HTTP API endpoint can be used as an activity which fits well with many REST/microservice setups.

## TODO

Types:

* `activity_options`
* `activity_ref`
* `child_workflow_options`
* `child_workflow_ref`
* `timer`
* `workflow_info`

Functions:

* `complete_with_continue_as_new`
* `emit_metric`
* `log_level`
* `set_query_handler`
* `set_search_attributes`
* `set_signal_handler`
* `signal_workflow`
* `start_activity`
* `start_child_workflow`
* `start_http_activity`
* `start_timer`
* More for waiting on results of activities and workflows