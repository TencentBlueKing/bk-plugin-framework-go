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
	SetFail(traceID string, err error) error
	SetSuccess(traceID string) error
	Version() string
}
