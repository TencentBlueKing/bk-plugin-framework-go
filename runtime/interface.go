package runtime

type ContextReader interface {
	ReadInputs(v interface{}) error
	ReadContextInputs(v interface{}) error
}

type ObjectStore interface {
	Write(traceID string, v interface{}) error
	Read(traceID string, v interface{}) error
}

type PluginExecuteRuntime interface {
	GetOutputsStore() ObjectStore
	GetContextStore() ObjectStore
	SetPoll(traceID string, version string, invokeCount int, interval int) error
}

type PluginScheduleExecuteRuntime interface {
	PluginExecuteRuntime
	SetFail(traceID string, err error) error
	SetSuccess(traceID string) error
}
