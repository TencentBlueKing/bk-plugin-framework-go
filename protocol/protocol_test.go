// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package protocol

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/info"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
)

var protocolTestVersionSeq uint64

type protocolTestPlugin struct {
	version string
	desc    string
}

func (p protocolTestPlugin) Version() string { return p.version }
func (p protocolTestPlugin) Desc() string    { return p.desc }
func (p protocolTestPlugin) Execute(ctx *kit.Context) error {
	return ctx.WriteOutputs(map[string]interface{}{"ok": true})
}

func nextProtocolTestVersion() string {
	return fmt.Sprintf("12.0.%d", atomic.AddUint64(&protocolTestVersionSeq, 1))
}

func TestResponseEnvelope(t *testing.T) {
	success := OK(map[string]interface{}{"value": 1})
	require.True(t, success.Result)
	require.Equal(t, 0, success.Code)
	require.Equal(t, "success", success.Message)
	require.Equal(t, map[string]interface{}{"value": 1}, success.Data)

	failure := Error(40404, "plugin not found")
	require.False(t, failure.Result)
	require.Equal(t, 40404, failure.Code)
	require.Equal(t, "plugin not found", failure.Message)
	require.Nil(t, failure.Data)
}

func TestBuildMetaUsesFrameworkProtocolContract(t *testing.T) {
	version := nextProtocolTestVersion()
	hub.MustInstallV2(protocolTestPlugin{version: version, desc: "meta plugin"}, hub.PluginSpec{})

	data := BuildMeta(MetaOptions{
		Code:           "new-go-plugin",
		Description:    "demo",
		RuntimeVersion: "1.0.0",
	})

	require.Equal(t, "new-go-plugin", data.Code)
	require.Equal(t, "demo", data.Description)
	require.Contains(t, data.Versions, version)
	require.Equal(t, "go", data.Language)
	require.Equal(t, info.Version(), data.FrameworkVersion)
	require.Equal(t, "1.0.0", data.RuntimeVersion)
	require.NotNil(t, data.AllowScope)

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"allow_scope":{}`)
}

func TestBuildDetailKeepsGoJSONSchemaOutOfRenderForm(t *testing.T) {
	version := nextProtocolTestVersion()
	hub.MustInstallV2(protocolTestPlugin{version: version, desc: "detail plugin"}, hub.PluginSpec{
		Inputs: struct {
			Mode string `json:"mode"`
		}{},
		Outputs: struct {
			OK bool `json:"ok"`
		}{},
		Form: []byte(`{"mode":{"component":"input"}}`),
	})

	data, err := BuildDetail(version, DetailOptions{EnablePluginCallback: true})
	require.NoError(t, err)
	require.Equal(t, version, data.Version)
	require.Equal(t, "detail plugin", data.Desc)
	require.True(t, data.EnablePluginCallback)
	require.Contains(t, data.Inputs["properties"], "mode")
	require.Contains(t, data.Outputs["properties"], "ok")
	require.Nil(t, data.Forms.RenderForm)

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"renderform":null`)
}

func TestBuildDetailKeepsLegacyInputsFormAsInputs(t *testing.T) {
	version := nextProtocolTestVersion()
	hub.MustInstall(protocolTestPlugin{version: version, desc: "legacy plugin"}, nil, nil, []byte(`{
		"type": "object",
		"properties": {
			"template_id": {
				"type": "number",
				"title": "模板 ID"
			}
		}
	}`))

	data, err := BuildDetail(version, DetailOptions{})
	require.NoError(t, err)
	require.Equal(t, "object", data.Inputs["type"])
	require.Contains(t, data.Inputs["properties"], "template_id")
	require.Nil(t, data.Forms.RenderForm)
}

func TestBuildDetailReturnsMissingVersionError(t *testing.T) {
	data, err := BuildDetail("99.99.99", DetailOptions{})
	require.Error(t, err)
	require.Empty(t, data.Version)
}
