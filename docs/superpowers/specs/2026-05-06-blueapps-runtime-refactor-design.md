# 基于 blueapps-go 的 Go 插件运行时重构设计

日期：2026-05-06

## 背景

当前 Go 插件框架实际由两部分组成：

- `bk-plugin-framework-go`：插件开发 SDK，提供 `kit.Plugin`、`kit.Context`、`hub.MustInstall`、`executor` 状态机等能力。
- `beego-runtime`：插件运行时，提供 Web 路由、APIGW 资源、worker、存储、部署入口和 Beego 相关逻辑。

这次重构的目标是废弃 `beego-runtime`，基于 `github.com/TencentBlueKing/blueapps-go` 重做 Go 插件运行时，让 Go 版本架构更接近 Python 版本的 `bk-plugin-framework` + `bk-plugin-runtime` 分层。

迁移目标采用兼容级别 B：

- 插件业务代码尽量不改。
- `main.go`、`go.mod`、`app_desc.yml` 等入口和部署配置允许少量调整。
- 不承诺兼容所有 `beego-runtime` 内部包和未文档化行为。

## 核心决策

拆成两个 Go module：

```text
bk-plugin-framework-go
bk-plugin-runtime-go
```

`bk-plugin-framework-go` 继续作为轻量、稳定的插件 SDK，不依赖 `blueapps-go`。

`bk-plugin-runtime-go` 是新的运行时 module，依赖 `bk-plugin-framework-go` 和 `blueapps-go`，负责启动入口、HTTP 协议、APIGW 同步、调度 worker、存储实现和 plugin API dispatch。

这个拆分的好处是：

- SDK 不被 Gin、GORM、Redis、APIGW、OTel 等运行时依赖污染。
- blueapps-go 升级影响主要收敛在 runtime module 内部。
- Go 版本和 Python 版本的分层模型保持一致。

## 仓库职责

### bk-plugin-framework-go

保持小而稳定：

```text
bk-plugin-framework-go/
  kit/
  hub/
  executor/
  runtime/
  constants/
```

职责：

- 定义 `kit.Plugin`。
- 定义 `kit.Context`。
- 通过 `hub` 注册插件版本。
- 生成和暴露插件元信息、schema 和 form。
- 通过 `executor` 执行插件状态机。
- 在 `runtime` 包中定义框架无关接口。

该 module 不应 import Gin、GORM、Redis、APIGW SDK 或 `blueapps-go`。

### bk-plugin-runtime-go

新 runtime module：

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

职责：

- 提供 `runner.Run()`。
- 通过 blueapps-go 公开 API 初始化配置、日志、DB、Redis、tracing、metrics 和 Gin runtime。
- 注册 `/bk_plugin/*` 插件协议路由。
- 持久化插件 schedule 状态。
- 运行 poll/callback 调度 worker。
- 校验 APIGW 调用身份和插件调用范围。
- 同步 APIGW 资源。
- 支持插件自定义 API 注册和 dispatch。

`internal/blueappsadapter` 是薄适配层，不复制 blueapps-go 代码。若 blueapps-go 缺少必要扩展点，优先给 blueapps-go 增加 hook 或公开 helper，而不是在插件 runtime 内部复制实现。

## 开发者迁移方式

迁移前：

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

迁移后：

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

理想情况下只需要把 runtime import 从：

```go
github.com/TencentBlueKing/beego-runtime/runner
```

改成：

```go
github.com/TencentBlueKing/bk-plugin-runtime-go/runner
```

插件业务代码继续实现：

```go
type Plugin interface {
    Version() string
    Desc() string
    Execute(ctx *kit.Context) error
}
```

既有 `kit.Context` 方法保持可用：

```go
ctx.ReadInputs(&inputs)
ctx.ReadContextInputs(&contextInputs)
ctx.WriteOutputs(outputs)
ctx.WaitPoll(after)
ctx.SetSuccess()
ctx.SetFail(err)
```

既有注册方式保持可用：

```go
hub.MustInstall(plugin, contextInputs, outputs, inputsForm)
```

新增更完整的注册方式用于后续补齐 Python 能力：

```go
hub.MustInstallV2(plugin, hub.PluginSpec{
    Inputs:        Inputs{},
    ContextInputs: ContextInputs{},
    Outputs:       Outputs{},
    Form:          form,
})
```

## 运行时命令

`bk-plugin-runtime-go/runner` 应尽量保留旧 runtime 的命令形态：

```text
server
worker
syncapigw
collectstatics
version
```

命令语义：

- `server`：启动基于 blueapps-go/Gin 的插件 HTTP 服务。
- `worker`：启动插件 schedule worker。
- `syncapigw`：同步插件 APIGW 资源。
- `collectstatics`：保留兼容命令；若 blueapps-go 不需要，则输出明确 no-op 提示。
- 无参数：默认启动 `server`，兼容本地开发习惯。

这样存量 `app_desc.yml` 可以尽量保持：

```yaml
processes:
  web:
    command: ./plugin server
  worker:
    command: ./plugin worker
```

## HTTP 协议

runtime 提供插件协议：

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

兼容原则：

- 保留旧 Go runtime 已文档化的响应字段。
- Python 兼容字段通过新增字段补齐，不替换旧字段。
- `trace_id`、状态值、outputs、错误信息保持稳定可见。
- 开发态调试路由必须可关闭，生产环境不能默认暴露敏感信息。

`meta` 返回 runtime 版本、语言和已注册插件版本。

`detail` 返回插件描述、schema 和 form：

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

`invoke` 创建一次 trace 并执行插件。

`schedule` 查询持久化执行状态和结果。

`callback` 接收外部系统回调数据，并恢复处于 callback 等待态的插件 trace。

## 状态机

旧 Go 状态继续有效：

```text
Empty -> Success
Empty -> Fail
Empty -> Poll -> Success
Empty -> Poll -> Fail
Empty -> Poll -> Poll
```

callback 能力扩展后：

```text
Empty    -> Callback
Poll     -> Callback
Callback -> Poll
Callback -> Success
Callback -> Fail
```

`Poll` 和 `Callback` 都是等待态，最终必须进入 `Success` 或 `Fail`。

SDK 可新增 callback 方法，但不能改变 `kit.Plugin` 接口：

```go
ctx.WaitCallback(callbackInfo)
ctx.ReadCallback(&payload)
```

不使用 callback 的旧插件不受影响。

## 调度和存储

插件 poll/callback 状态必须可恢复、可追踪、可重试，不能依赖进程内 goroutine 作为核心调度机制。

runtime 自己维护数据库 schedule 表：

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

索引：

- `trace_id` 唯一索引。
- `state + next_run_at + locked_until + finished_at` 组合索引，用于 worker 扫描。

worker 流程：

1. `invoke` 初次执行插件。
2. 插件调用 `WaitPoll(after)` 后，runtime 持久化 `StatePoll` 和 `next_run_at`。
3. worker 扫描到期记录。
4. worker 通过 `locked_until` 条件更新抢锁。
5. worker 调用 `executor.Schedule`。
6. runtime 持久化下一状态：`Success`、`Fail`、`Poll` 或 `Callback`。

如果 worker 崩溃，`locked_until` 过期后其他 worker 可以接管。

## callback 安全

callback token 不能简单等同于 `trace_id`。

token 要求：

- 包含 `trace_id`、`nonce`、`expire_at`。
- 使用 HMAC 或认证加密。
- 数据库只保存 token hash 或 nonce 元数据。
- 拒绝过期、伪造、重复使用、已完成 trace 或状态不匹配的 callback。

callback 流程：

1. 校验 token 签名和过期时间。
2. 根据 trace ID 读取 schedule。
3. 确认当前状态是 `Callback` 且未完成。
4. 写入 callback payload。
5. 通过 worker 恢复执行，不在 HTTP handler 内联继续执行插件。

重复 callback 请求应具备幂等性。

## APIGW 和权限

`invoke`、`schedule`、`plugin_api_dispatch` 必须完整校验 APIGW JWT：

- 校验签名。
- 校验网关来源。
- 解析调用方 app code。
- 解析 operator。
- 解析租户和请求元信息。
- 校验插件调用范围。
- 把 caller 信息写入 schedule 审计字段。

补齐类似 Python runtime 的 allow scope：

```go
hub.Configure(hub.Options{
    AllowScope: []string{"bk_sops", "bk_itsm"},
})
```

默认策略：

- 开发环境可允许本地免认证。
- 生产环境必须显式鉴权和范围校验。

## plugin API

插件自定义 API 应成为 runtime 的一等能力。

推荐公开 API：

```go
import "github.com/TencentBlueKing/bk-plugin-runtime-go/pluginapi"

func init() {
    pluginapi.Register(func(r pluginapi.Router) {
        r.GET("/tasks/:id", getTask)
        r.POST("/tasks", createTask)
    })
}
```

稳定 API 优先暴露窄 Router 接口，而不是直接暴露 Gin：

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

可提供可选 Gin adapter 给高级用户，但文档应推荐窄接口。

dispatch 流程：

1. 调用方请求 `/bk_plugin/plugin_api_dispatch`。
2. runtime 校验 APIGW JWT 和权限。
3. runtime 校验目标 path 在 `/bk_plugin/plugin_api/` 范围内。
4. 注入 caller app、operator、request ID、trace ID、tenant、language。
5. 转发到注册的 plugin API handler。
6. 返回 handler 响应。

## schema 和 form

保留 legacy 入口：

```go
hub.MustInstall(plugin, contextInputs, outputs, inputsForm)
```

新增显式 schema 入口：

```go
hub.MustInstallV2(plugin, hub.PluginSpec{
    Inputs:        Inputs{},
    ContextInputs: ContextInputs{},
    Outputs:       Outputs{},
    Form:          form,
})
```

旧 API 继续返回既有 input form 信息；新 API 支持显式 input schema，更适合对齐 Python runtime。

## Python 能力对齐范围

首版必须支持：

- 插件版本注册。
- `meta/detail/invoke/schedule`。
- input form metadata。
- context input 和 output schema。
- poll 调度。
- invoke/schedule 的 APIGW 鉴权。
- plugin API dispatch。
- runtime 日志和 trace ID。
- 多插件版本。

首版建议支持或预留：

- callback 等待态。
- 安全 callback token。
- allow scope。
- 统一错误格式。
- plugin finish callback。
- 开发态 debug 路由。

后续增强：

- 更完整 debug panel。
- 更接近 Python Pydantic 的复杂 schema 表达。
- 自动发现插件版本目录。
- form 静态资源管理。
- 更细粒度租户隔离。

## 错误处理

HTTP 层统一错误格式：

```json
{
  "code": 40000,
  "message": "invalid plugin inputs",
  "data": null
}
```

schedule 查询保留插件状态和插件错误：

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

错误分层：

- 插件业务错误。
- runtime 校验或序列化错误。
- 数据库、APIGW、scheduler 等基础设施错误。

runtime 错误日志必须包含 request ID 和 trace ID。

## 可观测性

复用 blueapps-go 日志、tracing 和 metrics 基础设施。

每次插件执行日志包含：

- `trace_id`
- `plugin_version`
- `state`
- `invoke_count`
- `caller_app`
- `operator`
- `request_id`
- `elapsed_ms`
- `error_code`

指标：

```text
plugin_invoke_total
plugin_schedule_total
plugin_schedule_duration_seconds
plugin_schedule_fail_total
plugin_callback_total
plugin_waiting_tasks
```

## 迁移工具

建议提供：

```text
docs/migration/beego-runtime-to-runtime-go.md
tools/check-migration
examples/legacy-compatible-plugin
examples/plugin-with-poll
examples/plugin-with-callback
examples/plugin-api
```

`tools/check-migration` 检查：

- 是否 import `github.com/TencentBlueKing/beego-runtime/runner`。
- 是否 import Beego 包。
- 是否 import `beego-runtime` 内部包。
- 是否使用 Beego controller 自定义 plugin API。
- `app_desc.yml` command 是否兼容。
- 是否依赖旧 debug panel 行为。

## 测试策略

### bk-plugin-framework-go

轻量 SDK/executor 测试：

- `kit.Context`。
- `hub.MustInstall`。
- schema 生成。
- 版本校验。
- 状态流转。
- 同步成功和失败。
- poll 行为。
- callback SDK 行为。

### bk-plugin-runtime-go

runtime 集成测试：

- `/bk_plugin/meta`。
- `/bk_plugin/detail/:version`。
- `/bk_plugin/invoke/:version`。
- `/bk_plugin/schedule/:trace_id`。
- `/bk_plugin/callback/:token`。
- `/bk_plugin/plugin_api_dispatch`。
- APIGW 鉴权成功和失败。
- scheduler 抢锁和重试。
- callback 幂等。

兼容 fixture：

```text
legacy-sync-plugin
legacy-poll-plugin
legacy-plugin-api
python-protocol-compatible-plugin
```

CI 需要测试 runtime 对 pinned blueapps-go 版本的兼容性，并定期检查较新 blueapps-go 版本。

## 发布节奏

Phase 1：兼容 runtime。

- 发布 `bk-plugin-runtime-go/runner`。
- 支持 legacy `hub.MustInstall`。
- 支持同步插件和 poll 插件。
- 保留 runner 命令形态。
- 提供迁移指南和 examples。

Phase 2：Python 能力对齐。

- 增加 callback 状态。
- 增加 allow scope。
- 完善 plugin API dispatch 鉴权。
- 增加 plugin finish callback。
- 改进 detail schema 和 form metadata。

Phase 3：废弃 beego-runtime。

- 标记 `beego-runtime` deprecated。
- 不再新增功能。
- 只保留严重 bug 和安全修复。
- 发布明确停止维护时间。

## 风险和缓解

风险：blueapps-go 扩展点不足。  
缓解：优先使用公开 API，缺少 hook 时向 blueapps-go 补扩展点，不复制内部代码。

风险：旧插件依赖未文档化的 Beego runtime 行为。  
缓解：提供迁移检查工具、fixture 兼容测试和明确的不兼容说明。

风险：APIGW 权限收紧影响现有调用方。  
缓解：开发环境允许本地 bypass，生产环境提供清晰错误码和 allow-scope 配置。

风险：scheduler 行为和旧 runtime 不一致。  
缓解：用兼容测试锁住 trace ID、状态值、响应结构和 retry 行为。

## 非目标

- 不保留 Beego runtime 依赖。
- 不把 blueapps-go 暴露成插件开发者 API。
- 不复制 blueapps-go 内部实现。
- 不要求存量插件重写业务逻辑。
- 不兼容未文档化的 `beego-runtime` 内部包用法。

## 首版实现默认值

- runtime module path：`github.com/TencentBlueKing/bk-plugin-runtime-go`。
- 新注册 API：`hub.MustInstallV2(plugin, hub.PluginSpec{...})`，同时保留 legacy `hub.MustInstall(...)`。
- APIGW 集成只放在 runtime module，不能给 `bk-plugin-framework-go` 增加 APIGW 依赖。
- callback 恢复：HTTP handler 只保存 payload，后续通过 worker 恢复执行。
- plugin API Router 首版包含 middleware 支持，避免用户为了常见鉴权/审计逻辑直接依赖 Gin。
