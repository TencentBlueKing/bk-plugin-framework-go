package executor

import (
	"github.com/homholueng/bk-plugin-framework-go/constants"
	"github.com/homholueng/bk-plugin-framework-go/hub"
	"github.com/homholueng/bk-plugin-framework-go/kit"
	"github.com/homholueng/bk-plugin-framework-go/runtime"
)

func Execute(traceID string, version string, reader runtime.ContextReader, runtime runtime.PluginExecuteRuntime) (constants.State, error) {
	// get plugin
	p, err := hub.GetPlugin(version)
	if err != nil {
		return constants.StateFail, err
	}

	// init context
	c := kit.NewContext(traceID, constants.StateEmpty, 1, reader, runtime.GetContextStore(), runtime.GetOutputsStore())

	// execute
	if err := p.Execute(c); err != nil {
		return constants.StateFail, err
	}

	// no poll request, execute success
	if !c.WaitingPoll() {
		return constants.StateSuccess, nil
	}

	if err := runtime.SetPoll(traceID, version, c.InvokeCount(), c.PollInterval()); err != nil {
		return constants.StateFail, nil
	}

	return constants.StatePoll, nil
}
