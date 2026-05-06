# 从 beego-runtime 迁移到 bk-plugin-runtime-go

## 目标

本文说明已有 Go 插件如何从 `beego-runtime` 迁移到基于 blueapps-go 的 `bk-plugin-runtime-go`。

Phase 1 的目标是优先兼容已有同步插件和 `WaitPoll` 轮询插件，尽量不改插件业务逻辑。

## 最小代码改动

修改 `main.go` 里的 runtime import：

```diff
- "github.com/TencentBlueKing/beego-runtime/runner"
+ "github.com/TencentBlueKing/bk-plugin-runtime-go/runner"
```

插件业务代码保持不变：

```go
func (p MyPlugin) Execute(ctx *kit.Context) error {
    return nil
}
```

旧的插件注册方式继续可用：

```go
hub.MustInstall(MyPlugin{}, ContextInputs{}, Outputs{}, inputsForm)
```

## 依赖调整

添加新的 runtime module：

```bash
go get github.com/TencentBlueKing/bk-plugin-runtime-go@v0.1.0
go mod tidy
```

`bk-plugin-framework-go` 仍然是插件开发 SDK，`bk-plugin-runtime-go` 只负责运行时。

## 进程命令

现有 `app_desc.yml` 里的命令形态保持兼容：

```yaml
processes:
  web:
    command: ./plugin server
  worker:
    command: ./plugin worker
```

Phase 1 runtime 支持这些命令：

- `server`：启动插件 HTTP 服务。
- `worker`：启动 poll 调度 worker。
- `syncapigw`：保留兼容命令，生产 APIGW 同步能力在后续阶段完善。
- `collectstatics`：兼容旧命令，在新 runtime 中是 no-op。
- `version`：输出 runtime 版本。

## Phase 1 支持范围

- 同步插件。
- 使用 `ctx.WaitPoll` 的轮询插件。
- `/bk_plugin/meta`。
- `/bk_plugin/detail/:version`。
- `/bk_plugin/invoke/:version`。
- `/bk_plugin/schedule/:trace_id`。
- 基于数据库持久化的 schedule 状态。

## 需要手动迁移的情况

以下用法不能自动无缝迁移：

- 插件代码直接 import Beego。
- 插件代码直接 import `beego-runtime` 内部包。
- 使用 Beego controller 实现自定义 plugin API。
- 依赖旧 debug panel 的页面细节。
- 依赖旧 runtime 未文档化的响应字段或副作用。

这些场景需要按新 runtime 的公开 API 逐步迁移。
