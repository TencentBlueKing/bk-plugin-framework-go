// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package executor

import (
	"fmt"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/TencentBlueKing/bk-plugin-framework-go/runtime"
)

type testReader struct{}

func (r testReader) ReadInputs(v interface{}) error {
	return nil
}

func (r testReader) ReadContextInputs(v interface{}) error {
	return nil
}

type testStore struct{}

func (s testStore) Write(traceID string, v interface{}) error {
	return nil
}

func (s testStore) Read(traceID string, v interface{}) error {
	return nil
}

type testRuntime struct {
	pollCalled    bool
	callbackCalled bool
	failCalled    bool
	successCalled bool
	pollErr       error
	callbackErr   error
	failErr       error
}

func (r *testRuntime) GetOutputsStore() runtime.ObjectStore {
	return testStore{}
}

func (r *testRuntime) GetContextStore() runtime.ObjectStore {
	return testStore{}
}

func (r *testRuntime) SetPoll(traceID string, version string, invokeCount int, after time.Duration) error {
	r.pollCalled = true
	return r.pollErr
}

func (r *testRuntime) SetCallback(traceID string, version string, invokeCount int, timeout time.Duration) error {
	r.callbackCalled = true
	return r.callbackErr
}

func (r *testRuntime) SetFail(traceID string, err error) error {
	r.failCalled = true
	return r.failErr
}

func (r *testRuntime) SetSuccess(traceID string) error {
	r.successCalled = true
	return nil
}

type panicPlugin struct {
	version string
}

func (p panicPlugin) Version() string { return p.version }
func (p panicPlugin) Desc() string    { return "panic plugin" }
func (p panicPlugin) Execute(c *kit.Context) error {
	panic("boom")
}

type waitPollPlugin struct {
	version string
}

func (p waitPollPlugin) Version() string { return p.version }
func (p waitPollPlugin) Desc() string    { return "wait poll plugin" }
func (p waitPollPlugin) Execute(c *kit.Context) error {
	c.WaitPoll(time.Second)
	return nil
}

type waitCallbackPlugin struct {
	version string
}

func (p waitCallbackPlugin) Version() string { return p.version }
func (p waitCallbackPlugin) Desc() string    { return "wait callback plugin" }
func (p waitCallbackPlugin) Execute(c *kit.Context) error {
	c.WaitCallback(time.Hour)
	return nil
}

func TestExecuteGetPluginError(t *testing.T) {

}

func TestExecuteExecuteError(t *testing.T) {

}

func TestExecuteNoPoll(t *testing.T) {

}

func TestExecuteSetPollError(t *testing.T) {

}

func TestExecuteSetPollSuccess(t *testing.T) {

}

func TestExecuteRecoverPluginPanic(t *testing.T) {
	hub.MustInstallV2(panicPlugin{version: "8.0.0"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{}

	state, err := Execute("trace-panic", "8.0.0", testReader{}, rt, log.WithFields(log.Fields{}))

	assert.Equal(t, constants.StateFail, state)
	assert.EqualError(t, err, "plugin execute panic: boom")
}

func TestScheduleGetPluginErrorReturnsAfterSetFail(t *testing.T) {
	rt := &testRuntime{}

	err := Schedule("trace-missing", "9.8.7", 2, testReader{}, rt, log.WithFields(log.Fields{}))

	assert.Error(t, err)
	assert.True(t, rt.failCalled)
	assert.False(t, rt.successCalled)
	assert.False(t, rt.pollCalled)
}

func TestScheduleRecoverPluginPanic(t *testing.T) {
	hub.MustInstallV2(panicPlugin{version: "8.0.1"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{}

	err := Schedule("trace-panic", "8.0.1", 2, testReader{}, rt, log.WithFields(log.Fields{}))

	assert.EqualError(t, err, "plugin schedule panic: boom")
	assert.True(t, rt.failCalled)
}

func TestScheduleRecoverPluginPanicReportsSetFailError(t *testing.T) {
	hub.MustInstallV2(panicPlugin{version: "8.0.2"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{failErr: fmt.Errorf("store down")}

	err := Schedule("trace-panic", "8.0.2", 2, testReader{}, rt, log.WithFields(log.Fields{}))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SetFail after Execute panic")
	assert.True(t, rt.failCalled)
}

func TestScheduleSetPollErrorReturnsOriginalAfterSetFail(t *testing.T) {
	hub.MustInstallV2(waitPollPlugin{version: "8.0.3"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{pollErr: fmt.Errorf("poll write failed")}

	err := Schedule("trace-poll-error", "8.0.3", 2, testReader{}, rt, log.WithFields(log.Fields{}))

	assert.EqualError(t, err, "poll write failed")
	assert.True(t, rt.pollCalled)
	assert.True(t, rt.failCalled)
}

func TestExecuteSetCallbackSuccess(t *testing.T) {
	hub.MustInstallV2(waitCallbackPlugin{version: "8.0.4"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{}

	state, err := Execute("trace-callback", "8.0.4", testReader{}, rt, log.WithFields(log.Fields{}))

	assert.NoError(t, err)
	assert.Equal(t, constants.StateCallback, state)
	assert.True(t, rt.callbackCalled)
}

func TestScheduleWithCallbackStateSetCallbackSuccess(t *testing.T) {
	hub.MustInstallV2(waitCallbackPlugin{version: "8.0.5"}, hub.PluginSpec{Form: []byte(`{}`)})
	rt := &testRuntime{}

	err := ScheduleWithState("trace-callback", "8.0.5", 2, constants.StateCallback, testReader{}, rt, log.WithFields(log.Fields{}))

	assert.NoError(t, err)
	assert.True(t, rt.callbackCalled)
}
