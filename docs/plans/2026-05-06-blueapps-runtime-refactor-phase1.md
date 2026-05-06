# 基于 blueapps-go 的插件运行时重构 Phase 1 实现计划

**目标：** 交付第一版可替代 `beego-runtime` 的 Go 插件运行时，优先支持同步插件和 `WaitPoll` 轮询插件，并尽量保持插件业务代码不变。

**架构：** `bk-plugin-framework-go` 保持轻量 SDK；新增 sibling module `bk-plugin-runtime-go` 作为基于 blueapps-go 的运行时。Phase 1 完成 SDK 注册兼容、runtime 命令入口、`meta/detail/invoke/schedule` HTTP 协议、持久化 schedule store、poll worker、迁移文档和 legacy fixture。

**技术栈：** Go 1.22、`github.com/TencentBlueKing/bk-plugin-framework-go`、`github.com/TencentBlueKing/blueapps-go`、Gin、GORM、SQLite 测试、MySQL 部署、Cobra、Logrus executor adapter。

---

## 范围

Phase 1 实现范围：

- 保留 legacy `hub.MustInstall(...)`。
- 新增 `hub.MustInstallV2(...)`，支持显式 input schema。
- 新建 module `github.com/TencentBlueKing/bk-plugin-runtime-go`。
- 提供 `runner.Run()`。
- 支持命令：`server`、`worker`、`syncapigw`、`collectstatics`、`version`。
- 支持路由：`GET /bk_plugin/meta`、`GET /bk_plugin/detail/:version`、`POST /bk_plugin/invoke/:version`、`GET /bk_plugin/schedule/:trace_id`。
- 实现数据库持久化 schedule store。
- 实现基于数据库抢锁的 poll worker。
- 提供迁移指南和 legacy-compatible 示例。

不在 Phase 1 实现：

- callback 状态机。
- 完整 APIGW JWT 校验。
- plugin finish callback。
- 生产级 APIGW 资源同步。
- plugin custom API dispatch。

这些能力后续单独拆计划实现。

## 文件结构

当前 SDK 仓库 `bk-plugin-framework-go` 修改：

- `hub/registry.go`：新增 `PluginSpec`、`MustInstallV2`、input schema 存储和 form accessor。
- `hub/registry_test.go`：新增 legacy 行为和 `MustInstallV2` 测试。
- `docs/migration/beego-runtime-to-runtime-go.md`：迁移指南。

新增 sibling runtime 仓库 `../bk-plugin-runtime-go`：

- `go.mod`：runtime module 依赖。
- `runner/runner.go`：公开运行时入口。
- `cmd/root.go`：命令根入口，无参数默认 `server`。
- `cmd/server.go`：启动 HTTP server。
- `cmd/worker.go`：启动 poll worker。
- `cmd/syncapigw.go`：Phase 1 兼容命令。
- `cmd/collectstatics.go`：Phase 1 no-op 兼容命令。
- `cmd/version.go`：输出 runtime 版本。
- `internal/version/version.go`：runtime 版本常量。
- `internal/blueappsadapter/bootstrap.go`：通过 blueapps-go 公开 API 初始化配置、日志、DB、Redis、缓存。
- `internal/httpx/response.go`：统一响应 envelope。
- `internal/store/model.go`：schedule 模型和接口。
- `internal/store/json.go`：JSON map 存储类型。
- `internal/store/gorm_store.go`：GORM schedule store。
- `internal/store/gorm_store_test.go`：store 测试。
- `internal/runtimeadapter/reader.go`：executor context reader。
- `internal/runtimeadapter/object_store.go`：context/outputs object store。
- `internal/runtimeadapter/execute_runtime.go`：executor runtime adapter。
- `internal/server/router.go`：插件协议路由。
- `internal/server/handlers.go`：`meta/detail/invoke/schedule` handler。
- `internal/server/handlers_test.go`：HTTP 协议测试。
- `internal/scheduler/worker.go`：DB-backed poll worker。
- `internal/scheduler/worker_test.go`：worker 测试。
- `examples/legacy-compatible-plugin/main.go`：legacy 兼容示例。
- `examples/legacy-compatible-plugin/inputs_form.json`：示例输入 form。
- `docs/migration/beego-runtime-to-runtime-go.md`：runtime 仓库迁移指南。

## 任务 1：增强 framework 注册 API

目标：

- 保留旧 `hub.MustInstall(plugin, contextInputs, outputs, inputsForm)` 行为。
- 新增 `hub.MustInstallV2(plugin, hub.PluginSpec{...})`。
- `PluginDetail` 新增 input schema 和 render form accessor。

实现要点：

- `PluginSpec` 包含 `Inputs`、`ContextInputs`、`Outputs`、`Form`。
- legacy `MustInstall` 继续把 `inputsForm` 作为旧版 input schema/form 返回，保证存量协议兼容。
- `MustInstallV2` 通过 Go struct 生成显式 input schema，同时保留 form metadata。

验证命令：

```bash
go test ./hub -count=1
go test ./... -count=1
```

提交：

```bash
git add hub/registry.go hub/registry_test.go
git commit -m "feat: add explicit plugin registration spec"
```

实际提交：

```text
f376514 feat: add explicit plugin registration spec
```

## 任务 2：创建 runtime module 骨架

目标：

- 在 `/Users/dengyh/Projects/bk-plugin-runtime-go` 创建新 module。
- 提供 `runner.Run()`。
- 提供兼容命令骨架。

实现要点：

- module path：`github.com/TencentBlueKing/bk-plugin-runtime-go`。
- Go 版本：`1.22`。
- runtime 依赖 `bk-plugin-framework-go` 和 `blueapps-go`。
- 本地开发通过 `replace github.com/TencentBlueKing/bk-plugin-framework-go => ../bk-plugin-framework-go` 指向当前 SDK 仓库。
- `cmd.Execute()` 在无参数时默认追加 `server`，保持本地开发体验。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go mod tidy
go test ./... -count=1
```

提交：

```bash
git add .
git commit -m "feat: add plugin runtime scaffold"
```

实际提交：

```text
0566abb feat: add plugin runtime scaffold
```

## 任务 3：添加 blueapps 初始化适配和响应工具

目标：

- runtime 通过 blueapps-go 公开 API 初始化基础设施。
- 提供统一 HTTP response envelope。

实现要点：

- `internal/blueappsadapter/bootstrap.go` 调用 blueapps-go 的 `config.Load`、`i18n.InitMsgMap`、`logging.InitLogger`、`database.InitDBClient`、`redis.InitRedisClient`、`memory.InitCache`。
- 不复制 blueapps-go 的初始化代码，只做运行时薄适配。
- `internal/httpx/response.go` 提供 `OK` 和 `Error`。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
```

该任务已合并在 runtime scaffold 提交中：

```text
0566abb feat: add plugin runtime scaffold
```

## 任务 4：实现持久化 schedule store

目标：

- 用数据库持久化插件执行状态。
- 支持 poll worker 抢锁和重试。

核心模型：

```text
Schedule
  trace_id
  plugin_version
  state
  invoke_count
  inputs
  context_inputs
  context_data
  outputs
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
```

实现要点：

- `JSONMap` 实现 `driver.Valuer` 和 `sql.Scanner`。
- `GormStore.Create` 自动填充空 JSON 字段。
- `ClaimDue` 只领取 `StatePoll`、未完成、已到期、未锁定或锁过期的任务。
- 领取任务时用条件更新 `locked_by/locked_until`，避免多 worker 重复执行。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./internal/store -count=1
```

提交：

```text
25ee338 feat: add durable schedule store and executor adapter
```

## 任务 5：实现 executor runtime adapter

目标：

- 把 framework 的 `runtime.ContextReader`、`runtime.ObjectStore`、`PluginExecuteRuntime`、`PluginScheduleExecuteRuntime` 映射到 runtime schedule store。

实现要点：

- `runtimeadapter.Reader` 从 schedule 的 `Inputs` 和 `ContextInputs` 读取数据。
- `ObjectStore` 将 `ctx.Write` 写入 `context_data`，将 `ctx.WriteOutputs` 写入 `outputs`。
- `ExecuteRuntime.SetPoll` 更新 `StatePoll` 和 `next_run_at`。
- `ExecuteRuntime.SetSuccess/SetFail` 更新最终状态。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./internal/runtimeadapter -count=1
```

提交：

```text
25ee338 feat: add durable schedule store and executor adapter
```

## 任务 6：实现插件 HTTP 协议

目标：

- 提供 `meta/detail/invoke/schedule` 四个核心协议路由。

路由：

```text
GET  /bk_plugin/meta
GET  /bk_plugin/detail/:version
POST /bk_plugin/invoke/:version
GET  /bk_plugin/schedule/:trace_id
```

实现要点：

- `meta` 返回语言、runtime version、插件版本列表。
- `detail` 返回插件版本、描述、inputs、context_inputs、outputs、`forms.renderform`。
- `invoke` 创建 `trace_id` 和 schedule 记录，调用 `executor.Execute`。
- 同步成功时标记 `StateSuccess`。
- 插件调用 `WaitPoll` 时通过 runtime adapter 标记 `StatePoll`。
- `schedule` 返回 trace 状态、outputs 和错误信息。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./internal/server -count=1
go test ./... -count=1
```

提交：

```text
5fec2ce feat: add plugin HTTP protocol and poll worker
```

## 任务 7：实现 poll worker

目标：

- worker 定期扫描到期 `StatePoll` 任务并恢复执行。

实现要点：

- `Worker.Run` 按 `Interval` 循环执行。
- `Worker.RunOnce` 调用 `ClaimDue` 抢锁。
- worker 调用 `executor.Schedule` 时传入 `item.InvokeCount + 1`，保持旧 `beego-runtime` 的重入语义。
- 插件执行后通过 runtime adapter 更新状态为 `Success`、`Fail` 或新的 `Poll`。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./internal/scheduler -count=1
go test ./... -count=1
```

提交：

```text
5fec2ce feat: add plugin HTTP protocol and poll worker
```

## 任务 8：添加迁移文档和 legacy fixture

目标：

- 给存量插件用户提供迁移说明。
- 提供一个只替换 runtime import 的 legacy-compatible 示例。

文档：

- `docs/migration/beego-runtime-to-runtime-go.md`
- `../bk-plugin-runtime-go/docs/migration/beego-runtime-to-runtime-go.md`

fixture：

```text
../bk-plugin-runtime-go/examples/legacy-compatible-plugin/
  main.go
  inputs_form.json
```

fixture 行为：

- 使用 legacy `hub.MustInstall(...)`。
- 使用 `ctx.WaitPoll(time.Second)`。
- 第二次调度时写入 outputs。
- `main()` 调用 `bk-plugin-runtime-go/runner.Run()`。

验证命令：

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
go test ./examples/legacy-compatible-plugin -count=1
```

提交：

```text
cc6cfe0 docs: add beego runtime migration guide
42a58ea docs: add beego runtime migration fixture
```

## 任务 9：端到端兼容 smoke test

目标：

- 确认 SDK 和 runtime 两个仓库都处于可构建、可测试状态。

验证命令：

```bash
cd ../bk-plugin-runtime-go/examples/legacy-compatible-plugin
go build -o /tmp/legacy-compatible-plugin .

cd ../bk-plugin-framework-go
go test ./... -count=1

cd ../bk-plugin-runtime-go
go test ./... -count=1
```

当前环境缺少 Go 工具链：

```text
zsh: command not found: go
zsh: command not found: gofmt
```

因此本轮只能完成代码落地和静态文件检查。后续在安装 Go 1.22 的环境中必须补跑：

```bash
cd /Users/dengyh/Projects/bk-plugin-framework-go
gofmt -w hub/registry.go hub/registry_test.go
go test ./... -count=1

cd /Users/dengyh/Projects/bk-plugin-runtime-go
gofmt -w $(find . -name '*.go' -not -path './.git/*')
go mod tidy
go test ./... -count=1
go test ./examples/legacy-compatible-plugin -count=1
```

## 任务 10：最终 review 修复

最终代码审查发现并修复了以下问题：

- 插件 `panic` 时，`executor.Execute` 和 `executor.Schedule` 现在会 recover 并返回错误，避免 trace 卡在未完成状态或 worker 进程直接退出。
- `executor.Schedule` 获取插件版本失败后，在 `SetFail` 成功后会立即返回，不再继续执行空插件实例。
- `GormStore.ClaimDue` 二次抢锁时重新约束 `state = StatePoll`、`finished_at IS NULL`、`next_run_at <= now`，避免 stale candidate 把已完成任务重新锁住并重复执行。
- `MarkSuccess/MarkFail` 会同步保存当前 `invoke_count`，避免终态审计数据偏小。
- `executor.Schedule` 在 `SetPoll` 失败且成功标记失败后，会返回原始 `SetPoll` 错误，避免 worker 吞掉本次调度写入失败。

实际提交：

```text
529122f fix: harden executor error handling
e63d174 fix: harden schedule locking and invoke count
c4dc66d fix: return schedule poll errors
```

## 后续计划

Phase 1 完成后，后续建议按独立计划继续：

1. callback 状态机和安全 token。
2. 完整 APIGW JWT 校验和 allow scope。
3. plugin API dispatch。
4. plugin finish callback。
5. debug/logs 接口和更完整开发调试体验。
6. 迁移检查工具 `tools/check-migration`。

## 本轮已知限制

- `syncapigw` 目前是兼容命令，生产同步逻辑未实现。
- `collectstatics` 是 no-op。
- APIGW 鉴权暂未接入。
- callback 暂未实现。
- plugin API dispatch 暂未实现。
- 当前环境缺少 Go 工具链，无法生成 `go.sum`、无法 `gofmt`、无法跑测试。
