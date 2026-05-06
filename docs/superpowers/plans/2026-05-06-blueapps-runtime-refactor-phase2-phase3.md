# 基于 blueapps-go 的插件运行时重构 Phase 2/3 实现计划

> **给 agentic workers：** 按任务逐步执行；每个行为变更先补测试，再实现。当前环境缺少 `go`/`gofmt`，本轮只能提交代码和静态检查，真实测试需在 Go 1.22 环境补跑。

**目标：** 在 Phase 1 兼容 runtime 基础上补齐 callback、allow scope、finish callback、plugin API dispatch，并完成 `beego-runtime` 弃用标记。

**架构：** `bk-plugin-framework-go` 只增加 SDK 能力和 hub 配置，不依赖 blueapps-go。`bk-plugin-runtime-go` 实现 callback token、callback HTTP endpoint、worker 恢复、轻量 allow-scope 认证、finish callback 通知和 plugin API dispatch。`beego-runtime` 只做弃用文档，不新增运行时代码。

**技术栈：** Go 1.22、Gin、GORM、HMAC-SHA256 callback token、`net/http` finish callback、SDK optional interface。

---

## 任务 1：SDK callback 与 hub 配置

文件：

- 修改 `runtime/interface.go`
- 修改 `kit/context.go`
- 修改 `kit/context_test.go`
- 修改 `executor/execute.go`
- 修改 `executor/schedule.go`
- 修改 `hub/registry.go`
- 修改 `hub/registry_test.go`

实现：

- `kit.Context` 新增 `WaitCallback(timeout time.Duration)`、`WaitingCallback()`、`CallbackTimeout()`、`ReadCallback(v interface{})`。
- `runtime` 新增 optional interfaces：`CallbackReader`、`PluginCallbackRuntime`。
- `executor.Execute/Schedule` 在插件等待 callback 时调用 runtime 的 `SetCallback`。
- `executor.Schedule` 支持以 `StateCallback` 重新进入插件。
- `hub.Configure` / `hub.GetOptions` 支持 `AllowScope` 和 `EnablePluginCallback`。

测试：

```bash
go test ./kit ./hub ./executor -count=1
```

## 任务 2：runtime callback token、store 和 callback endpoint

文件：

- 新增 `internal/callback/token.go`
- 新增 `internal/callback/token_test.go`
- 修改 `internal/store/model.go`
- 修改 `internal/store/gorm_store.go`
- 修改 `internal/store/gorm_store_test.go`
- 修改 `internal/runtimeadapter/reader.go`
- 修改 `internal/runtimeadapter/execute_runtime.go`
- 修改 `internal/server/router.go`
- 修改 `internal/server/handlers.go`
- 修改 `internal/server/handlers_test.go`
- 修改 `internal/scheduler/worker.go`

实现：

- schedule 表增加 callback payload、token hash、过期时间、接收时间、callback URL。
- `SetCallback` 持久化 `StateCallback` 和安全 token。
- callback token 使用 HMAC-SHA256 签名，payload 包含 trace ID、过期时间、nonce。
- `POST /bk_plugin/callback/:token` 验证 token、写入 callback payload，并让 worker 可恢复执行。
- worker 同时领取 `StatePoll` 和已收到 payload 的 `StateCallback`。

测试：

```bash
go test ./internal/callback ./internal/store ./internal/server ./internal/scheduler -count=1
```

## 任务 3：allow scope 和 finish callback

文件：

- 新增 `internal/auth/scope.go`
- 新增 `internal/finishcallback/notifier.go`
- 修改 `internal/server/router.go`
- 修改 `internal/server/handlers.go`
- 修改 `internal/scheduler/worker.go`
- 修改 `internal/store/model.go`

实现：

- 若 `hub.Options.AllowScope` 非空，`invoke` / `schedule` / `plugin_api_dispatch` 需要请求头 `X-Bkapi-App-Code` 命中 allow scope。
- 开发环境或未配置 allow scope 时默认放行。
- 从 context inputs 的 `plugin_callback_info.url` / `token` 读取 finish callback 配置。
- 插件最终 `Success` / `Fail` 后发送 finish callback。
- finish callback 失败只记录错误，不回滚插件终态。

测试：

```bash
go test ./internal/auth ./internal/finishcallback ./internal/server ./internal/scheduler -count=1
```

## 任务 4：plugin API dispatch

文件：

- 新增 `pluginapi/router.go`
- 新增 `internal/pluginapi/registry.go`
- 修改 `internal/server/router.go`
- 新增 `internal/server/plugin_api.go`
- 新增 `internal/server/plugin_api_test.go`

实现：

- 提供 `pluginapi.RegisterGin(func(gin.IRouter))` 作为 Phase 2 过渡 API。
- runtime 注册 `/bk_plugin/plugin_api/*path`。
- runtime 注册 `/bk_plugin/plugin_api_dispatch`，只允许转发到 `/bk_plugin/plugin_api/` 范围内。
- dispatch 注入调用方 header：`X-Bkapi-App-Code`、`X-Bkapi-Operator`、`X-Bkapi-Request-Id`。

测试：

```bash
go test ./pluginapi ./internal/server -count=1
```

## 任务 5：Phase 3 弃用 beego-runtime

文件：

- 修改 `/Users/dengyh/Projects/beego-runtime/README.md`
- 新增 `/Users/dengyh/Projects/beego-runtime/docs/deprecation.md`
- 修改 `docs/migration/beego-runtime-to-runtime-go.md`

实现：

- 在 `beego-runtime` README 顶部标记 deprecated。
- 指向 `bk-plugin-runtime-go` 和迁移指南。
- 说明只保留严重 bug / 安全修复，不再新增能力。

测试：

```bash
git -C /Users/dengyh/Projects/beego-runtime diff --check
```

## 最终验证

在 Go 1.22 环境补跑：

```bash
cd /Users/dengyh/Projects/bk-plugin-framework-go
gofmt -w $(find kit runtime executor hub -name '*.go')
go test ./... -count=1

cd /Users/dengyh/Projects/bk-plugin-runtime-go
gofmt -w $(find . -name '*.go' -not -path './.git/*')
go mod tidy
go test ./... -count=1

cd /Users/dengyh/Projects/beego-runtime
git diff --check
```
