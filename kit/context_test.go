package kit

import (
	"testing"

	"github.com/homholueng/bk-plugin-framework-go/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestState(t *testing.T) {
	assert.Equal(t, constants.StateEmpty, constants.State(1))
	assert.Equal(t, constants.StatePoll, constants.State(2))
	assert.Equal(t, constants.StateCallback, constants.State(3))
	assert.Equal(t, constants.StateSuccess, constants.State(4))
	assert.Equal(t, constants.StateFail, constants.State(5))
}

type MockContextReader struct {
	mock.Mock
}

func (r *MockContextReader) ReadInputs(v interface{}) error {
	r.Called(v)
	return nil
}

func (r *MockContextReader) ReadContextInputs(v interface{}) error {
	r.Called(v)
	return nil
}

type MockStore struct {
	mock.Mock
}

func (s *MockStore) Write(traceID string, v interface{}) error {
	s.Called(traceID, v)
	return nil
}

func (s *MockStore) Read(traceID string, v interface{}) error {
	s.Called(traceID, v)
	return nil
}

func TestContext(t *testing.T) {
	var v interface{}

	reader := MockContextReader{}
	reader.On("ReadInputs", &v).Return(nil)
	reader.On("ReadContextInputs", &v).Return(nil)

	outputsStore := MockStore{}
	outputsStore.On("Write", "trace", &v).Return(nil)
	outputsStore.On("Read", "trace", &v).Return(nil)

	store := MockStore{}
	store.On("Write", "trace", &v).Return(nil)
	store.On("Read", "trace", &v).Return(nil)

	c := Context{
		traceID:      "trace",
		state:        constants.StateEmpty,
		pollInterval: -1,
		invokeCount:  1,
		reader:       &reader,
		store:        &store,
		outputsStore: &outputsStore,
	}

	assert.Equal(t, c.TraceID(), "trace")
	assert.Equal(t, c.State(), constants.StateEmpty)
	assert.Equal(t, c.InvokeCount(), 1)

	// WaitPoll test
	assert.False(t, c.WaitingPoll())
	c.WaitPoll(1)
	assert.Equal(t, c.pollInterval, 1)
	assert.True(t, c.WaitingPoll())

	// Read test
	c.ReadInputs(&v)
	reader.AssertCalled(t, "ReadInputs", &v)

	// ReadContext test
	c.ReadContextInputs(&v)
	reader.AssertCalled(t, "ReadContextInputs", &v)

	// Write test
	c.Write(&v)
	store.AssertCalled(t, "Write", "trace", &v)

	// Read test
	c.Read(&v)
	store.AssertCalled(t, "Read", "trace", &v)

	// WriteOutputs test
	c.WriteOutputs(&v)
	outputsStore.AssertCalled(t, "Write", "trace", &v)

	// ReadOutputs test
	c.ReadOutputs(&v)
	outputsStore.AssertCalled(t, "Read", "trace", &v)
}

func TestNewContext(t *testing.T) {
	traceID := "trace"
	state := constants.StateEmpty
	invokeCount := 1
	reader := MockContextReader{}
	outputsStore := MockStore{}
	store := MockStore{}

	c := NewContext(
		traceID,
		state,
		invokeCount,
		&reader,
		&store,
		&outputsStore,
	)

	assert.Equal(t, c.traceID, traceID)
	assert.Equal(t, c.state, state)
	assert.Equal(t, c.pollInterval, -1)
	assert.Equal(t, c.invokeCount, invokeCount)
	assert.Equal(t, c.reader, &reader)
	assert.Equal(t, c.store, &store)
	assert.Equal(t, c.outputsStore, &outputsStore)
}
