// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package kit collect the basic tool for developer to
// develop a bk-plugin.
package kit

import (
	"time"

	"github.com/homholueng/bk-plugin-framework-go/constants"
	"github.com/homholueng/bk-plugin-framework-go/runtime"
)

// A Context store all context information and data for once plugin execution.
type Context struct {
	traceID      string
	state        constants.State
	pollInterval time.Duration
	waitingPoll  bool
	invokeCount  int
	reader       runtime.ContextReader
	store        runtime.ObjectStore
	outputsStore runtime.ObjectStore
}

// NewContext returns a new Context instance.
//
// The traceID set the unique id of these execution.
//
// The state set the current state of these execution.
//
// The invokeCount set the times of Plugin instance Execute method called.
//
// The reader set the read source of inputs.
//
// The store set the store of context data.
//
// The outputsStore set the store of plugin outputs.
func NewContext(traceID string, state constants.State, invokeCount int, reader runtime.ContextReader, store runtime.ObjectStore, ouputsStore runtime.ObjectStore) *Context {
	return &Context{
		traceID:      traceID,
		state:        state,
		invokeCount:  invokeCount,
		reader:       reader,
		store:        store,
		outputsStore: ouputsStore,
	}
}

// TraceID returns context trace id.
func (c *Context) TraceID() string {
	return c.traceID
}

// TraceID returns current state of once execution.
func (c *Context) State() constants.State {
	return c.state
}

// InvokeCount returns invoke count of Plugin instance Execute method called.
func (c *Context) InvokeCount() int {
	return c.invokeCount
}

// PollInterval returns next poll execute's interval.
func (c *Context) PollInterval() time.Duration {
	return c.pollInterval
}

// WaitPoll tells executor to execute plugin with poll state after duration.
func (c *Context) WaitPoll(interval time.Duration) {
	c.pollInterval = interval
	c.waitingPoll = true
}

// WaitingPoll returns whether current execution should enter poll state.
func (c *Context) WaitingPoll() bool {
	return c.waitingPoll
}

// ReadInputs parses inputs data and store the result
// in the value pointed to by v.
func (c *Context) ReadInputs(v interface{}) error {
	return c.reader.ReadInputs(v)
}

// ReadContextInputs parses context inputs data and store the result
// in the value pointed to by v.
func (c *Context) ReadContextInputs(v interface{}) error {
	return c.reader.ReadContextInputs(v)
}

// Write will store the value pointed to by v to context data.
func (c *Context) Write(v interface{}) error {
	return c.store.Write(c.traceID, v)
}

// Read parses context data data and store the result
// in the value pointed to by v.
func (c *Context) Read(v interface{}) error {
	return c.store.Read(c.traceID, v)
}

// Write will store the value pointed to by v to outputs.
func (c *Context) WriteOutputs(v interface{}) error {
	return c.outputsStore.Write(c.traceID, v)
}

// ReadOutputs parses outputs data and store the result
// in the value pointed to by v.
func (c *Context) ReadOutputs(v interface{}) error {
	return c.outputsStore.Read(c.traceID, v)
}
