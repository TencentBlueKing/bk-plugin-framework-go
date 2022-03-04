// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package executor provide the Execute and Schedule action
// for bk-plugin execution model.
package executor

import (
	"github.com/homholueng/bk-plugin-framework-go/constants"
	"github.com/homholueng/bk-plugin-framework-go/hub"
	"github.com/homholueng/bk-plugin-framework-go/kit"
	"github.com/homholueng/bk-plugin-framework-go/runtime"
	log "github.com/sirupsen/logrus"
)

// Execute define the execute action for bk-plugin execution model.
//
// The traceID represent the unique id for this execution.
//
// The version represent the version of plugin which will be executed.
//
// The reader set the read source of inputs.
//
// The runtime set the execute runtime use in execute action.
func Execute(traceID string, version string, reader runtime.ContextReader, runtime runtime.PluginExecuteRuntime, logger *log.Entry) (constants.State, error) {
	// get plugin
	p, err := hub.GetPlugin(version)
	if err != nil {
		logger.Errorf("get plugin failed: %v\n", err)
		return constants.StateFail, err
	}

	// init context
	c := kit.NewContext(traceID, constants.StateEmpty, 1, reader, runtime.GetContextStore(), runtime.GetOutputsStore(), logger)

	// execute
	if err := p.Execute(c); err != nil {
		logger.Errorf("plugin execute return err: %v\n", err)
		return constants.StateFail, err
	}

	// no poll request, execute success
	if !c.WaitingPoll() {
		return constants.StateSuccess, nil
	}

	if err := runtime.SetPoll(traceID, version, c.InvokeCount(), c.PollInterval()); err != nil {
		logger.Errorf("execute success but set poll err: %v\n", err)
		return constants.StateFail, err
	}

	return constants.StatePoll, nil
}
