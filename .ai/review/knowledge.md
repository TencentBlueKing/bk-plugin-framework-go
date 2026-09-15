# bk-plugin-framework-go 审查知识库

## 核验基线与使用方式

- 2026-09-09 核验远程 `main`：`d4e2018958d5dfe47d4e6235324fb40ea2f609e5`。这是知识快照；审查时以 PR 的 base/head、实际依赖和调用者源码为准。
- 根 `go.mod` 声明 Go 1.17；本仓库是插件 SDK 和项目生成模板，不是插件 SaaS，也不拥有生产 HTTP server、数据库连接池或 worker 实现。
- 事实优先级：当前实现与有效依赖 > 对应测试 > 文档。根 `README.md` 包含旧 Beego/其他语言的说明；`docs/migration/beego-runtime-to-runtime-go.md` 和 `template/{{cookiecutter.project_name}}/README.md` 提供迁移、生成项目说明，但也要与源码对照。
- 历史事故、未合并 PR、其他分支的实现只用于定位审查入口，不能据此断言当前仍有同一问题。

## 源码地图与跨仓库职责

| 入口 | 当前职责与审查重点 |
| --- | --- |
| `kit/plugin.go`、`kit/context.go` | `Plugin` 的 `Version/Desc/Execute` 接口；一次调用的 trace ID、状态、调用次数、输入/上下文、持久化上下文及输出；poll/callback 请求。 |
| `hub/registry.go` | 版本注册、重复版本校验、schema 反射、旧 `MustInstall` 与 `MustInstallV2`、进程级 `AllowScope` 和完成回调配置。 |
| `protocol/meta.go`、`protocol/detail.go`、`protocol/response.go` | 标准元信息、详情与响应封装；字段名、类型、空值和旧调用方兼容。 |
| `runtime/interface.go` | 存储、执行、调度与可选 callback 能力接口；实现由独立 runtime 提供。 |
| `executor/execute.go`、`executor/schedule.go`、`executor/callback.go` | 初次执行及恢复执行，调用插件后选择 poll/callback/终态，处理插件 error/panic 和持久化错误。 |
| `pluginapi/router.go`、`pluginapi/params.go` | 基于 `net/http` 的插件自定义 API 注册与路径参数；实际路由和鉴权由 runtime 适配，不能引入对 Gin/Beego 的业务依赖。 |
| `template/cookiecutter.json`、`template/{{cookiecutter.project_name}}/` | 可发布插件项目的独立依赖图、启动注册、内嵌表单、样例插件、PaaS 描述和 APIGW 同步入口。 |

独立仓库 `bk-plugin-runtime-go` 的 `internal/server/`、`internal/store/`、`internal/scheduler/`、`internal/blueappsadapter/` 分别承接 HTTP 协议、数据库状态、worker、平台配置/连接池。涉及这些能力的 SDK 修改，要核对该 runtime **实际锁定版本**的适配接口；SDK 测试替身并不能证明生产持久化成功。

## 表单、注册与依赖兼容

1. `hub.MustInstall(plugin, contextInputs, outputs, inputsForm)` 保持旧调用形状，传入的 JSON 表单直接成为 `inputs` schema，默认不输出为 `forms.renderform`。`MustInstallV2` 从 `PluginSpec.Inputs/ContextInputs/Outputs` 生成 schema，并解析 `PluginSpec.Form`。
2. 本次 main 的 `PluginSpec` 只有 `Inputs`、`ContextInputs`、`Outputs`、`Form`；没有 `RenderForm` 字段。`BuildDetail` 优先使用 `DetailOptions.RenderForm`；未指定时，V2 的非空 `Form` 解析为 JSON 对象并输出至 `forms.renderform`，旧注册路径为 `null`。依据：`hub/registry.go`、`protocol/detail.go`。
3. 当前模板仅嵌入 `versions/v100/form.json`，没有 `form.js`。以后涉及 JS RenderForm 时，必须分别核对 JSON Schema 的 `inputs`、原始 JS 字符串的 `forms.renderform`、注册 key、默认值/必填规则和消费者实际渲染方式；不能因为字段同名就宣称兼容。与 Python/BK-SOPS 对齐时读取目标版本消费者，包括 JS 注册标识是否使用插件 app code，以及 `hookable` 等属性在哪条渲染路径生效。
4. 当前 `template/cookiecutter.json` 默认 framework `v1.0.4`、runtime `v0.2.8`，生成模块使用 Go 1.23.0。独立 runtime 在本次核验的 main `3db91f59466149a4c87ab808b9ada1971698d431` 中自报 `v0.2.9`；模板 pin 不等于远程最新 runtime，也不代表所有已部署插件已经升级。
5. 模板 `go.mod` 带有公开替换 `github.com/TencentBlueKing/gopkg v1.3.0 => github.com/TencentBlueKing/gopkg v1.0.9`，用于当前 runtime/APIGW SDK 依赖组合。Go 依赖模块的 `replace` 不会自动传给最终消费模块；更改 pin 或替换时要验证生成项目的有效依赖和 `go.sum`，不能只在 SDK checkout 里构建。
6. 给公开结构体增加字段、改变方法签名或 schema 反射会影响下游；同时检查带字段名和不带字段名的结构体字面量。新增必填输入、改变输出类型或重写已发布插件版本语义，需要明确迁移/新版本策略。

## 执行状态与协议契约

- `constants/state.go` 固定值：`EMPTY=1`、`POLL=2`、`CALLBACK=3`、`SUCCESS=4`、`FAIL=5`。`executor.Execute` 从 EMPTY、invokeCount=1 开始；`ScheduleWithState` 接收 runtime 恢复的状态与次数。数值是外部协议的一部分。
- `kit.Context` 用同一 `TraceID()` 访问 `ContextData` 和 `Outputs`，`ReadInputs` 与 `ReadContextInputs` 是两类输入；request ID、插件 trace ID、业务 task ID 不可混用。框架没有把任意 `tenant_id` 自动变成租户鉴权。
- `WaitPoll` / `WaitCallback` 参数是 `time.Duration`，调用方须明确 `time.Second` 等单位。插件正常返回且没有等待请求才进入 SUCCESS；callback 分支目前优先于 poll；error/panic 走 FAIL。等待状态仍需 runtime 的 `SetPoll/SetCallback` 成功落库。
- `PrepareCallback` 通过可选 `PluginCallbackPrepareRuntime` 提前取得地址，`ReadCallback` 通过可选 `CallbackReader` 读取回调数据。缺少能力应保留明确错误；不要给基础 runtime 接口强加破坏旧适配器的必选方法。
- `executor/schedule.go` 已覆盖找不到版本、panic、`SetPoll` 失败及 `SetFail` 二次失败等分支。审查应防止错误被吞掉、失败后继续成功写入、原始错误丢失；不能把已修复的历史分支重新当作现存缺陷。
- 对接链路是调用方 → runtime `/bk_plugin/invoke/:version` → SDK executor → 插件；GET `/bk_plugin/schedule/:trace_id` 查询已持久化结果，实际恢复执行由 worker 完成；POST `/bk_plugin/callback/:token` 接收第三方回调。完成通知调用方的 finish callback 是另一方向。HTTP 路径、请求 `inputs/context` 和状态查询兼容由 runtime 验证。
- SDK 不提供“业务请求恰好执行一次”的保证。外部调用、重复 invoke、poll 恢复、callback 重投与存储失败的组合必须按实际 runtime 行为分析；盲目重试插件 `Execute` 可能重复创建外部任务。

## 生成项目与验证入口

`template/{{cookiecutter.project_name}}/main.go` 使用 `MustInstallV2` 注册 `versions/v100` 后调用 runtime `runner.Run()`；默认插件仅在 EMPTY 下同步把 `hello` 写入 `world`。`app_desc.yml` 使用 PaaS `specVersion: 3`，包含 web/worker、MySQL/Redis 及执行 `bin/sync_apigateway.sh` 的 preRelease hook。脚本会实际同步 APIGW、获取公钥；审查无需执行此类外部写入。

| 变更区域 | 现有测试与适用验证 |
| --- | --- |
| 注册、schema、表单、协议封装 | `hub/registry_test.go` 的 `TestMustInstallLegacyKeepsInputsFormAsInputsSchema` / `TestMustInstallV2StoresExplicitSchemasAndForm`；`protocol/protocol_test.go` 的旧表单、V2 表单、meta、envelope 用例。 |
| 执行与恢复 | `executor/execute_test.go` 的 panic、找不到版本、持久化失败、callback/准备 callback 用例；`kit/context_test.go`、`constants/state_test.go`。该执行测试文件中有数个空测试函数，不能把测试名称当作覆盖证据。 |
| 自定义 API 抽象 | `pluginapi/router_test.go` 的注册副本、路径参数用例；实际 dispatch 还需 runtime `internal/server/plugin_api_test.go`。 |
| 模板 | 渲染到临时目录后执行 `go test -mod=readonly ./... -count=1` 和 `go build -mod=readonly ./...`；样例断言在生成的 `versions/v100/plugin_test.go`。根模块测试不会递归覆盖这个独立 Go 模块。 |
| 跨 runtime 行为 | 有效 runtime 版本的同步/poll/callback 协议用例，涉及存储时追加目标 MySQL 的集成证据。模板中的新 SDK 符号必须存在于正式 pin 的模块中。 |

2026-09-09 在上述 SDK 基线运行 `GOWORK=off go test -mod=readonly ./... -count=1` 通过。本次知识库核验没有渲染模板、访问真实 BK-SOPS 或运行 MySQL 集成验证。日后的 PR 应报告本次实际执行的检查，不沿用这份快照的“通过”。
