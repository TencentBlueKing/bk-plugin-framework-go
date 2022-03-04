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
	log "github.com/sirupsen/logrus"

	"github.com/homholueng/bk-plugin-framework-go/constants"
	"github.com/homholueng/bk-plugin-framework-go/hub"
	"github.com/homholueng/bk-plugin-framework-go/kit"
	"github.com/homholueng/bk-plugin-framework-go/runtime"

	"github.com/pkg/errors"
)

// Schedule define the schedule action for bk-plugin execution model.
//
// The traceID represent the unique id for this execution.
//
// The version represent the version of plugin which will be executed.
//
// The reader set the read source of inputs.
//
// The runtime set the execute runtime use in schedule action.
func Schedule(traceID string, version string, invokeCount int, reader runtime.ContextReader, runtime runtime.PluginScheduleExecuteRuntime, logger *log.Entry) error {
	// get plugin
	p, err := hub.GetPlugin(version)
	if err != nil {
		if setErr := runtime.SetFail(traceID, err); setErr != nil {
			return errors.Wrap(errors.Wrap(err, setErr.Error()), "SetFail after GetPlugin error")
		}
	}

	// init context
	c := kit.NewContext(traceID, constants.StatePoll, invokeCount, reader, runtime.GetContextStore(), runtime.GetOutputsStore(), logger)

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
