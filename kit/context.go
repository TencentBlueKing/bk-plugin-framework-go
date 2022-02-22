package kit

import (
	"github.com/homholueng/bk-plugin-framework-go/constants"
	"github.com/homholueng/bk-plugin-framework-go/runtime"
)

type Context struct {
	traceID      string
	state        constants.State
	pollInterval int
	invokeCount  int
	reader       runtime.ContextReader
	store        runtime.ObjectStore
	outputsStore runtime.ObjectStore
}

func NewContext(traceID string, state constants.State, invokeCount int, reader runtime.ContextReader, store runtime.ObjectStore, ouputsStore runtime.ObjectStore) *Context {
	return &Context{
		traceID:      traceID,
		state:        state,
		pollInterval: -1,
		invokeCount:  invokeCount,
		reader:       reader,
		store:        store,
		outputsStore: ouputsStore,
	}
}

func (c *Context) TraceID() string {
	return c.traceID
}

func (c *Context) State() constants.State {
	return c.state
}

func (c *Context) InvokeCount() int {
	return c.invokeCount
}

func (c *Context) PollInterval() int {
	return c.pollInterval
}

func (c *Context) WaitPoll(interval int) {
	if interval < 0 {
		c.pollInterval = 0
	} else {
		c.pollInterval = interval
	}
}

func (c *Context) WaitingPoll() bool {
	return c.pollInterval >= 0
}

func (c *Context) ReadInputs(v interface{}) error {
	return c.reader.ReadInputs(v)
}

func (c *Context) ReadContextInputs(v interface{}) error {
	return c.reader.ReadContextInputs(v)
}

func (c *Context) Write(v interface{}) error {
	return c.store.Write(c.traceID, v)
}

func (c *Context) Read(v interface{}) error {
	return c.store.Read(c.traceID, v)
}

func (c *Context) WriteOutputs(v interface{}) error {
	return c.outputsStore.Write(c.traceID, v)
}

func (c *Context) ReadOutputs(v interface{}) error {
	return c.outputsStore.Read(c.traceID, v)
}
