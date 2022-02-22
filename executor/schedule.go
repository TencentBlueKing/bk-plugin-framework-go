package executor

import (
	"bpfgo/constants"
	"bpfgo/hub"
	"bpfgo/kit"
	"bpfgo/runtime"

	"github.com/pkg/errors"
)

func Schedule(traceID string, version string, invokeCount int, reader runtime.ContextReader, runtime runtime.PluginExecuteRuntime) error {
	// get plugin
	p, err := hub.GetPlugin(version)
	if err != nil {
		if setErr := runtime.SetFail(traceID, err); setErr != nil {
			return errors.Wrap(errors.Wrap(err, setErr.Error()), "SetFail after GetPlugin error")
		}
	}

	// init context
	c := kit.NewContext(traceID, constants.StatePoll, invokeCount, reader, runtime.GetContextStore(), runtime.GetOutputsStore())

	// execute
	if err := p.Execute(c); err != nil {
		if setErr := runtime.SetFail(traceID, err); setErr != nil {
			return errors.Wrap(errors.Wrap(err, setErr.Error()), "SetFail after Execute error")
		}
	}

	// no poll request, execute success
	if !c.WaitingPoll() {
		if err := runtime.SetSuccess(traceID); err != nil {
			return err
		}
		return nil
	}

	if err := runtime.SetPoll(traceID, version, c.InvokeCount(), c.PollInterval()); err != nil {
		if setErr := runtime.SetFail(traceID, err); setErr != nil {
			return errors.Wrap(errors.Wrap(err, setErr.Error()), "SetFail after SetPoll error")
		}
	}

	return nil
}
