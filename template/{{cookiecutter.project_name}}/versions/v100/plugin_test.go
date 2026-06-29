package v100

import (
	"encoding/json"
	"testing"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/sirupsen/logrus"
)

type testReader struct {
	inputs        map[string]interface{}
	contextInputs map[string]interface{}
}

func (r testReader) ReadInputs(v interface{}) error {
	return marshalTo(r.inputs, v)
}

func (r testReader) ReadContextInputs(v interface{}) error {
	return marshalTo(r.contextInputs, v)
}

type testStore struct {
	data map[string][]byte
}

func newTestStore() *testStore {
	return &testStore{data: map[string][]byte{}}
}

func (s *testStore) Write(traceID string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.data[traceID] = data
	return nil
}

func (s *testStore) Read(traceID string, v interface{}) error {
	return json.Unmarshal(s.data[traceID], v)
}

func marshalTo(src interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func TestHelloWorldPluginWritesOutput(t *testing.T) {
	contextStore := newTestStore()
	outputsStore := newTestStore()
	reader := testReader{
		inputs: map[string]interface{}{
			"hello": "world",
		},
		contextInputs: map[string]interface{}{
			"executor": "admin",
		},
	}

	plugin := &Plugin{}
	c := kit.NewContext("trace-hello", constants.StateEmpty, 1, reader, contextStore, outputsStore, logrus.NewEntry(logrus.StandardLogger()))
	if err := plugin.Execute(c); err != nil {
		t.Fatalf("execute plugin: %v", err)
	}
	if c.WaitingPoll() || c.WaitingCallback() {
		t.Fatalf("default plugin should finish synchronously")
	}

	var outputs Outputs
	if err := outputsStore.Read("trace-hello", &outputs); err != nil {
		t.Fatalf("read outputs: %v", err)
	}
	if outputs.World != "world" {
		t.Fatalf("World = %q, want %q", outputs.World, "world")
	}
}

func TestHelloWorldPluginRejectsUnsupportedState(t *testing.T) {
	contextStore := newTestStore()
	outputsStore := newTestStore()
	reader := testReader{
		inputs:        map[string]interface{}{"hello": "world"},
		contextInputs: map[string]interface{}{"executor": "admin"},
	}

	plugin := &Plugin{}
	c := kit.NewContext("trace-poll", constants.StatePoll, 2, reader, contextStore, outputsStore, logrus.NewEntry(logrus.StandardLogger()))
	if err := plugin.Execute(c); err == nil {
		t.Fatalf("expected unsupported state error")
	}
}
