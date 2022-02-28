// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package runtime define the plugin runtime related interfaces.
package runtime

// ContextReader is the interface that wraps the basic read method
// used by Context
//
// ReadInputs should parses inputs data and store the result
// in the value pointed to by v.
//
// ReadContextInputs should parses context inputs data and store the result
// in the value pointed to by v.
type ContextReader interface {
	ReadInputs(v interface{}) error
	ReadContextInputs(v interface{}) error
}

// ObjectStore is the interface that wraps the basic store operate method.
//
// Write should store the value pointed to by v with traceID
//
// Read should parses data with traceID and store the result
// in the value pointed to by v.
type ObjectStore interface {
	Write(traceID string, v interface{}) error
	Read(traceID string, v interface{}) error
}

// PluginExecuteRuntime is the interface that wraps the basic runtime method
// used in plugin execute phase.
//
// GetOutputsStore retruns a ObjectStore for plugin outputs storage.
//
// GetContextStore retruns a ObjectStore for plugin context data storage.
//
// SetPoll send a poll request to runtime. after that, runtime should execute
// the Plugin's Execute method of this version with the traceID and invokeCount in context.
type PluginExecuteRuntime interface {
	GetOutputsStore() ObjectStore
	GetContextStore() ObjectStore
	SetPoll(traceID string, version string, invokeCount int, interval int) error
}

// PluginExecuteRuntime is the interface that wraps the basic runtime method
// used in plugin schedule phase.
//
// SetFail should mark execution with traceID as StateFail because of err.
//
// SetSuccess should mark execution with traceID as StateSuccess.
type PluginScheduleExecuteRuntime interface {
	PluginExecuteRuntime
	SetFail(traceID string, err error) error
	SetSuccess(traceID string) error
}
