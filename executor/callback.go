package executor

import (
	"time"

	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	pluginruntime "github.com/TencentBlueKing/bk-plugin-framework-go/runtime"
)

func setCallbackPreparer(c *kit.Context, traceID string, version string, runtime pluginruntime.PluginExecuteRuntime) {
	callbackRuntime, ok := runtime.(pluginruntime.PluginCallbackPrepareRuntime)
	if !ok {
		return
	}
	c.SetCallbackPreparer(func(timeout time.Duration) (pluginruntime.CallbackPreparation, error) {
		return callbackRuntime.PrepareCallback(traceID, version, c.InvokeCount(), timeout)
	})
}
