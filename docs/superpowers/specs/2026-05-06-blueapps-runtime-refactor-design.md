# Blueapps-Based Go Plugin Runtime Refactor Design

Date: 2026-05-06

## Background

The current Go plugin framework is split across two practical concerns:

- `bk-plugin-framework-go` provides the plugin SDK concepts such as `kit.Plugin`, `kit.Context`, `hub.MustInstall`, and executor state transitions.
- `beego-runtime` provides the web runtime, APIGW resources, worker process, storage, and Beego-based deployment entrypoint.

The goal of this refactor is to retire `beego-runtime` and rebuild the Go plugin runtime on top of `github.com/TencentBlueKing/blueapps-go`, similar to how the Python plugin framework depends on the BlueKing application framework.

The migration target is compatibility level B:

- Existing plugin business code should generally remain unchanged.
- `main.go`, `go.mod`, and deployment configuration may require small changes.
- Full source-level compatibility with every `beego-runtime` internal API is not required.

## Design Decision

Use two Go modules:

```text
bk-plugin-framework-go
bk-plugin-runtime-go
```

`bk-plugin-framework-go` remains the stable SDK and framework-neutral execution layer. It must not depend on `blueapps-go`.

`bk-plugin-runtime-go` is the new runtime module. It depends on `bk-plugin-framework-go` and `blueapps-go`, and provides the executable runtime entrypoint, HTTP protocol, APIGW synchronization, scheduler, storage implementation, and plugin API dispatch.

This mirrors the Python split:

```text
bk-plugin-framework   -> development SDK
bk-plugin-runtime     -> BlueKing application runtime
```

## Repository Responsibilities

### bk-plugin-framework-go

This module should remain small and stable:

```text
bk-plugin-framework-go/
  kit/
  hub/
  executor/
  runtime/
  constants/
```

Responsibilities:

- Define `kit.Plugin`.
- Define `kit.Context`.
- Register plugin versions through `hub`.
- Generate and expose plugin metadata and schemas.
- Execute plugin state transitions in `executor`.
- Define framework-neutral runtime interfaces in `runtime`.

It should avoid importing Gin, GORM, Redis, APIGW SDKs, or `blueapps-go`.

### bk-plugin-runtime-go

This new module owns the runtime implementation:

```text
bk-plugin-runtime-go/
  runner/
  pluginapi/
  internal/
    blueappsadapter/
    server/
    scheduler/
    store/
    apigw/
    auth/
    callback/
```

Responsibilities:

- Provide `runner.Run()`.
- Initialize blueapps-go configuration, logging, database, Redis, tracing, metrics, and Gin runtime through blueapps-go public APIs.
- Register `/bk_plugin/*` protocol routes.
- Persist plugin schedule state.
- Run poll and callback scheduling workers.
- Verify APIGW authentication and plugin call scope.
- Synchronize APIGW resources.
- Provide plugin custom API registration and dispatch.

The `blueappsadapter` package is a thin adapter. It must not copy blueapps-go implementation code. If blueapps-go lacks an extension point, prefer contributing a hook or exported helper to blueapps-go rather than duplicating its internals.

## Developer Migration

### Before

```go
package main

import (
    "github.com/TencentBlueKing/beego-runtime/runner"
    "github.com/TencentBlueKing/bk-plugin-framework-go/hub"
)

func init() {
    hub.MustInstall(MyPlugin{}, ContextInputs{}, Outputs{}, inputsForm)
}

func main() {
    runner.Run()
}
```

### After

```go
package main

import (
    "github.com/TencentBlueKing/bk-plugin-runtime-go/runner"
    "github.com/TencentBlueKing/bk-plugin-framework-go/hub"
)

func init() {
    hub.MustInstall(MyPlugin{}, ContextInputs{}, Outputs{}, inputsForm)
}

func main() {
    runner.Run()
}
```

The ideal migration is a single runtime import change:

```diff
- github.com/TencentBlueKing/beego-runtime/runner
+ github.com/TencentBlueKing/bk-plugin-runtime-go/runner
```

Most plugin business code should continue to use:

```go
type Plugin interface {
    Version() string
    Desc() string
    Execute(ctx *kit.Context) error
}
```

Existing methods remain supported:

```go
ctx.ReadInputs(&inputs)
ctx.ReadContextInputs(&contextInputs)
ctx.WriteOutputs(outputs)
ctx.WaitPoll(after)
ctx.SetSuccess()
ctx.SetFail(err)
```

Existing registration remains supported:

```go
hub.MustInstall(plugin, contextInputs, outputs, inputsForm)
```

Newer APIs may be added for richer schemas and Python parity, but existing plugins should not be forced to migrate to them.

## Runtime Commands

`bk-plugin-runtime-go/runner` should preserve the old command shape where possible:

```text
server
worker
syncapigw
collectstatics
version
```

Command behavior:

- `server`: start the blueapps-go based Gin web runtime.
- `worker`: start the plugin schedule worker.
- `syncapigw`: synchronize plugin APIGW resources.
- `collectstatics`: preserve as a compatibility command. If not needed by blueapps-go, make it a no-op with an explicit message.
- no argument: start `server` for local developer convenience.

This keeps existing `app_desc.yml` commands close to their current form:

```yaml
processes:
  web:
    command: ./plugin server
  worker:
    command: ./plugin worker
```

## HTTP Protocol

The runtime provides the plugin protocol:

```text
GET  /bk_plugin/meta
GET  /bk_plugin/detail/:version
POST /bk_plugin/invoke/:version
GET  /bk_plugin/schedule/:trace_id

POST /bk_plugin/plugin_api_dispatch
ANY  /bk_plugin/plugin_api/*path

POST /bk_plugin/callback/:token
GET  /bk_plugin/logs/:trace_id
```

Compatibility rules:

- Preserve old Go runtime response fields where documented.
- Add Python-compatible fields as additive fields, not replacements.
- Keep `trace_id`, plugin state, outputs, and error visibility stable.
- Keep development-friendly routes optional and gated for non-production environments.

`meta` returns framework version, runtime version, language, and registered plugin versions.

`detail` returns version description, schemas, and form metadata:

```json
{
  "version": "1.0.0",
  "desc": "...",
  "inputs": {},
  "context_inputs": {},
  "outputs": {},
  "forms": {
    "renderform": {}
  }
}
```

`invoke` creates a trace and executes the plugin once.

`schedule` reads the persisted execution state and result.

`callback` receives external callback data and resumes a callback-waiting plugin trace.

## State Machine

The old Go states stay valid:

```text
Empty -> Success
Empty -> Fail
Empty -> Poll -> Success
Empty -> Poll -> Fail
Empty -> Poll -> Poll
```

Callback support extends the model:

```text
Empty    -> Callback
Poll     -> Callback
Callback -> Poll
Callback -> Success
Callback -> Fail
```

`Poll` and `Callback` are waiting states. Every trace eventually finishes as `Success` or `Fail`.

The framework SDK may add callback methods without changing `kit.Plugin`:

```go
ctx.WaitCallback(callbackInfo)
ctx.ReadCallback(&payload)
```

Existing plugins that do not use callbacks remain unaffected.

## Scheduling and Storage

Plugin poll and callback state must be durable. Do not rely on blueapps-go's in-process async goroutines as the core plugin scheduler.

Use a DB-backed schedule table owned by `bk-plugin-runtime-go`.

Suggested table:

```text
plugin_schedules
  id
  trace_id
  plugin_version
  state
  invoke_count
  inputs
  context_inputs
  context_data
  outputs
  callback_data
  error_code
  error_message
  error_detail
  next_run_at
  locked_by
  locked_until
  finished_at
  caller_app
  operator
  request_id
  tenant_id
  created_at
  updated_at
```

Indexing:

- unique index on `trace_id`
- composite index on `state`, `next_run_at`, `locked_until`, `finished_at`

Worker flow:

1. `invoke` executes the plugin.
2. If the plugin calls `WaitPoll(after)`, runtime persists state `Poll` and `next_run_at`.
3. Worker scans due records.
4. Worker claims a record using conditional update on `locked_until`.
5. Worker calls `executor.Schedule`.
6. Runtime persists the next state: `Success`, `Fail`, `Poll`, or `Callback`.

If a worker crashes, `locked_until` eventually expires and another worker can retry the trace.

## Callback Security

Callback token handling must not treat token as plain `trace_id`.

Token requirements:

- Include `trace_id`, `nonce`, and `expire_at`.
- Use HMAC or authenticated encryption.
- Store only token hash or nonce metadata in the database.
- Reject expired, malformed, reused, completed, or state-mismatched callbacks.

Callback handling:

1. Verify token signature and expiry.
2. Load schedule by trace ID.
3. Ensure current state is `Callback` and trace is unfinished.
4. Store callback payload.
5. Mark callback data ready and resume execution through scheduler.

Repeated callback requests should be idempotent: return an already-processed result without changing finished traces.

## APIGW and Authorization

`invoke`, `schedule`, and `plugin_api_dispatch` must fully verify APIGW JWTs:

- Verify signature.
- Verify gateway source.
- Parse caller app code.
- Parse operator username.
- Parse tenant and request metadata where available.
- Enforce plugin call scope.
- Store caller information in schedule audit fields.

Add an allow-scope configuration to align with Python runtime behavior:

```go
hub.Configure(hub.Options{
    AllowScope: []string{"bk_sops", "bk_itsm"},
})
```

Default policy:

- Development mode may allow local unauthenticated calls.
- Production mode should require explicit authentication and scope validation.

## Plugin API

Plugin custom API should be a first-class capability in the runtime.

Recommended public API:

```go
import "github.com/TencentBlueKing/bk-plugin-runtime-go/pluginapi"

func init() {
    pluginapi.Register(func(r pluginapi.Router) {
        r.GET("/tasks/:id", getTask)
        r.POST("/tasks", createTask)
    })
}
```

Expose a narrow router interface rather than raw Gin as the primary API:

```go
type Router interface {
    GET(path string, h HandlerFunc)
    POST(path string, h HandlerFunc)
    PUT(path string, h HandlerFunc)
    DELETE(path string, h HandlerFunc)
    Any(path string, h HandlerFunc)
    Group(path string, middlewares ...Middleware) Router
}
```

An optional Gin adapter may exist for advanced users, but the stable API should not require plugin authors to depend on Gin directly.

Dispatch flow:

1. Caller requests `/bk_plugin/plugin_api_dispatch`.
2. Runtime verifies APIGW JWT and authorization.
3. Runtime validates target path is under `/bk_plugin/plugin_api/`.
4. Runtime injects caller app, operator, request ID, trace ID, tenant, and language context.
5. Runtime forwards to registered plugin API handler.
6. Handler response is returned to the caller.

## Schema and Form Compatibility

Keep the legacy registration entrypoint:

```go
hub.MustInstall(plugin, contextInputs, outputs, inputsForm)
```

Add a richer API for new plugins:

```go
hub.MustInstallV2(plugin, hub.PluginSpec{
    Inputs:        Inputs{},
    ContextInputs: ContextInputs{},
    Outputs:       Outputs{},
    Form:          form,
})
```

The old API should continue generating context input and output schemas, and returning existing input form information.

The new API can add explicit input schema support and better Python parity.

## Python Parity Scope

Initial version must support:

- plugin version registration
- `meta`, `detail`, `invoke`, and `schedule`
- input form metadata
- context input and output schemas
- poll scheduling
- APIGW authentication for invoke and schedule
- plugin API dispatch
- runtime logs and trace IDs
- multiple plugin versions

Initial version should also support or prepare for:

- callback waiting state
- secure callback tokens
- allow scope
- unified error format
- plugin finish callback
- development debug routes

Later enhancements:

- richer debug panel
- richer schema semantics comparable to Python Pydantic behavior
- automatic plugin version discovery
- static form asset management
- finer tenant isolation rules

## Error Handling

Use a consistent HTTP error envelope:

```json
{
  "code": 40000,
  "message": "invalid plugin inputs",
  "data": null
}
```

Schedule responses should include trace state and plugin error information:

```json
{
  "code": 0,
  "message": "OK",
  "data": {
    "trace_id": "...",
    "state": 5,
    "outputs": {},
    "error": {
      "code": "PLUGIN_EXECUTE_ERROR",
      "message": "..."
    }
  }
}
```

Error categories:

- plugin business errors
- runtime validation or serialization errors
- infrastructure errors such as database, APIGW, or scheduler failures

Runtime errors should be logged with request ID and trace ID.

## Observability

Use blueapps-go logging, tracing, and metrics infrastructure.

Every plugin execution log should include:

- `trace_id`
- `plugin_version`
- `state`
- `invoke_count`
- `caller_app`
- `operator`
- `request_id`
- `elapsed_ms`
- `error_code`

Metrics:

```text
plugin_invoke_total
plugin_schedule_total
plugin_schedule_duration_seconds
plugin_schedule_fail_total
plugin_callback_total
plugin_waiting_tasks
```

## Migration Tooling

Provide:

```text
docs/migration/beego-runtime-to-runtime-go.md
tools/check-migration
examples/legacy-compatible-plugin
examples/plugin-with-poll
examples/plugin-with-callback
examples/plugin-api
```

`tools/check-migration` should detect:

- imports of `github.com/TencentBlueKing/beego-runtime/runner`
- imports of Beego packages
- imports of `beego-runtime` internal packages
- custom plugin API Beego controllers
- incompatible `app_desc.yml` commands
- likely reliance on old debug panel behavior

## Testing Strategy

### bk-plugin-framework-go

Run lightweight SDK and executor tests:

- `kit.Context`
- `hub.MustInstall`
- schema generation
- version validation
- state transition behavior
- sync success and failure
- poll scheduling behavior
- callback SDK behavior when added

### bk-plugin-runtime-go

Run runtime integration tests:

- `/bk_plugin/meta`
- `/bk_plugin/detail/:version`
- `/bk_plugin/invoke/:version`
- `/bk_plugin/schedule/:trace_id`
- `/bk_plugin/callback/:token`
- `/bk_plugin/plugin_api_dispatch`
- APIGW authentication success and failure
- scheduler locking and retry
- callback idempotency

Compatibility fixtures:

```text
legacy-sync-plugin
legacy-poll-plugin
legacy-plugin-api
python-protocol-compatible-plugin
```

CI should test `bk-plugin-runtime-go` against the pinned blueapps-go version and periodically check compatibility with newer blueapps-go releases.

## Release Plan

Phase 1: compatibility runtime

- Publish `bk-plugin-runtime-go/runner`.
- Support legacy `hub.MustInstall`.
- Support sync and poll plugins.
- Preserve runner command shape.
- Provide migration guide and examples.

Phase 2: Python parity

- Add callback state.
- Add allow scope.
- Complete plugin API dispatch authorization.
- Add plugin finish callback.
- Improve detail schema and form metadata.

Phase 3: beego-runtime deprecation

- Mark `beego-runtime` deprecated.
- Stop adding features to it.
- Keep only critical bug and security fixes for a defined period.
- Publish an end-of-maintenance timeline.

## Risks and Mitigations

Risk: blueapps-go does not expose enough extension points.

Mitigation: use public APIs where possible and contribute missing hooks upstream instead of copying blueapps-go internals.

Risk: old plugins depend on undocumented Beego runtime behavior.

Mitigation: provide migration checker, fixture-based compatibility tests, and explicit unsupported behavior docs.

Risk: stricter APIGW authorization breaks existing callers.

Mitigation: provide development-mode bypass, clear production errors, and allow-scope configuration.

Risk: scheduler behavior differs from old runtime.

Mitigation: lock old sync and poll behavior with compatibility tests around trace ID, state values, response shape, and retry behavior.

## Non-Goals

- Do not keep Beego as a runtime dependency.
- Do not expose blueapps-go as the plugin developer API.
- Do not copy blueapps-go implementation code into the plugin runtime.
- Do not require existing plugins to rewrite business logic for the initial migration.
- Do not guarantee compatibility for undocumented `beego-runtime` internal package usage.

## Implementation Defaults

Use these defaults for the first implementation plan:

- Runtime module path: `github.com/TencentBlueKing/bk-plugin-runtime-go`.
- New registration API: introduce `hub.MustInstallV2(plugin, hub.PluginSpec{...})` while keeping legacy `hub.MustInstall(...)`.
- APIGW integration: use BlueKing APIGW SDK APIs from the runtime module only; do not add APIGW dependencies to `bk-plugin-framework-go`.
- Callback resumption: store callback payload first, then resume through the worker. Do not execute plugin callback continuation inline in the HTTP callback handler.
- Plugin API router: include middleware support in the first stable router interface so common auth, audit, and request adaptation logic does not require exposing raw Gin.
