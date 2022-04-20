// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package hub

import (
	"encoding/json"
	"testing"

	"github.com/homholueng/bk-plugin-framework-go/kit"

	"github.com/alecthomas/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestEmptySchema(t *testing.T) {
	expected := []byte(`{"type": "object", "properties": {}, "required": [], "definitions": {}}`)
	assert.Equal(t, emptySchema, expected)
}

func TestVersionRe(t *testing.T) {
	var cases = []struct {
		in       string
		expected bool
	}{
		{"1.1.1", true},
		{"1.1.1rc", true},
		{"1.1.1beta", true},
		{"1", false},
		{"1.1", false},
		{"1.1.n", false},
		{"n.1.1", false},
		{"1.n.1", false},
	}

	for _, c := range cases {
		if actual := versionRe.MatchString(c.in); actual != c.expected {
			assert.Equal(t, actual, c.expected)
		}
	}
}

// PluginDetail test
type MetaTestPlugin struct{}

func (t *MetaTestPlugin) Version() string {
	return "1.1.1"
}

func (t *MetaTestPlugin) Desc() string {
	return "Desc"
}

func (t *MetaTestPlugin) Execute(c *kit.Context) error {
	return nil
}

func TestPluginDetailPlugin(t *testing.T) {
	plugin := MetaTestPlugin{}
	inputsSchema := []byte("{\"inputsSchema\": 1}")
	contextInputsSchema := []byte("{\"contextInputsSchema\": 2}")
	outputsSchema := []byte("{\"outputsSchema\": 3}")
	inputsSchemaJSON := map[string]interface{}{"inputsSchema": 1}
	contextInputsSchemaJSON := map[string]interface{}{"contextInputsSchema": 2}
	outputsSchemaJSON := map[string]interface{}{"outputsSchema": 3}

	detail := PluginDetail{
		plugin:                  &plugin,
		inputsSchema:            inputsSchema,
		contextInputsSchema:     contextInputsSchema,
		outputsSchema:           outputsSchema,
		inputsSchemaJSON:        inputsSchemaJSON,
		contextInputsSchemaJSON: contextInputsSchemaJSON,
		outputsSchemaJSON:       outputsSchemaJSON,
	}

	assert.Equal(t, detail.Plugin(), &plugin)
	assert.Equal(t, detail.InputsSchema(), inputsSchema)
	assert.Equal(t, detail.ContextInputsSchema(), contextInputsSchema)
	assert.Equal(t, detail.OutputsSchema(), outputsSchema)
	assert.Equal(t, detail.InputsSchemaJSON(), inputsSchemaJSON)
	assert.Equal(t, detail.ContextInputsSchemaJSON(), contextInputsSchemaJSON)
	assert.Equal(t, detail.OutputsSchemaJSON(), outputsSchemaJSON)
}

func TestReflectJSONSchema(t *testing.T) {
	// case 1
	type ReflectStruct struct {
		TemplateID int    `json:"template_id"`
		TaskName   string `json:"task_name"`
	}
	var rs ReflectStruct
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema, err := reflector.Reflect(&rs).MarshalJSON()
	assert.Nil(t, err)

	var schemaJSON map[string]interface{}
	err = json.Unmarshal(schema, &schemaJSON)
	assert.Nil(t, err)

	// case 2
	var emptySchemaJSON map[string]interface{}
	err = json.Unmarshal(emptySchema, &emptySchemaJSON)
	assert.Nil(t, err)

	// case 3
	type ReflectStructWithFrom struct {
		TemplateID int    `json:"template_id"`
		TaskName   string `json:"task_name"`
	}
	inputsForm := kit.Form{
		"template_id": {
			"attr1": "val1",
			"attr2": "val2",
		},
		"task_name": {
			"attr3": kit.F{
				"sub_attr3": "val3",
			},
		},
	}
	var rsf ReflectStructWithFrom
	rsfSchema, err := reflector.Reflect(&rsf).MarshalJSON()
	assert.Nil(t, err)

	var rsfSchemaJSON map[string]interface{}
	err = json.Unmarshal(schema, &rsfSchemaJSON)
	assert.Nil(t, err)
	properties := rsfSchemaJSON["properties"].(map[string]interface{})
	for prop, attrs := range inputsForm {
		for k, v := range attrs {
			property := properties[prop].(map[string]interface{})
			property[k] = v
		}
	}

	rsfSchema, err = json.Marshal(rsfSchemaJSON)
	assert.Nil(t, err)

	var cases = []struct {
		in                 interface{}
		extraAttrs         kit.Form
		expectedSchema     []byte
		expectedSchemaJSON interface{}
	}{
		{rs, nil, schema, schemaJSON},
		{rsf, inputsForm, rsfSchema, rsfSchemaJSON},
		{nil, nil, emptySchema, emptySchemaJSON},
	}

	for _, c := range cases {
		actualSchema, actualSchemaJSON, err := reflectJSONSchema(c.in, c.extraAttrs)
		assert.Nil(t, err)
		assert.Equal(t, c.expectedSchema, actualSchema)
		assert.Equal(t, c.expectedSchemaJSON, actualSchemaJSON)
	}
}

type MustInstallTestPlugin struct {
	version string
}

func (t *MustInstallTestPlugin) Version() string {
	return t.version
}

func (t *MustInstallTestPlugin) Desc() string {
	return "Desc"
}

func (t *MustInstallTestPlugin) Execute(c *kit.Context) error {
	return nil
}

type MustInstallTestPluginInput struct{}
type MustInstallTestPluginContextInput struct{}
type MustInstallTestPluginOutput struct{}

var InputsForm kit.Form = kit.Form{
	"template_id": {
		"attr1": "val1",
		"attr2": "val2",
	},
	"task_name": {
		"attr3": kit.F{
			"sub_attr3": "val3",
		},
	},
}

func TestMustInstall(t *testing.T) {
	clearHub()
	var success_cases = []struct {
		plugin        *MustInstallTestPlugin
		inputs        interface{}
		contextInputs interface{}
		outputs       interface{}
		inputsForm    kit.Form
	}{
		{&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil, nil},
		{&MustInstallTestPlugin{version: "1.0.1"}, MustInstallTestPluginInput{}, nil, nil, nil},
		{&MustInstallTestPlugin{version: "1.0.2"}, nil, MustInstallTestPluginContextInput{}, nil, nil},
		{&MustInstallTestPlugin{version: "1.0.3"}, nil, nil, MustInstallTestPluginOutput{}, nil},
		{&MustInstallTestPlugin{version: "1.0.4"}, MustInstallTestPluginInput{}, MustInstallTestPluginContextInput{}, MustInstallTestPluginOutput{}, nil},
		{&MustInstallTestPlugin{version: "1.0.5"}, MustInstallTestPluginInput{}, MustInstallTestPluginContextInput{}, MustInstallTestPluginOutput{}, InputsForm},
	}

	for _, c := range success_cases {
		assert.NotPanics(t, func() { MustInstall(c.plugin, c.inputs, c.contextInputs, c.outputs, c.inputsForm) }, "success case %v failed", c)
	}

	var panic_cases = []struct {
		plugin        *MustInstallTestPlugin
		inputs        interface{}
		contextInputs interface{}
		outputs       interface{}
	}{
		{&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil},
		{&MustInstallTestPlugin{version: "1.0"}, nil, nil, nil},
	}

	for _, c := range panic_cases {
		assert.Panics(t, func() { MustInstall(c.plugin, c.inputs, c.contextInputs, c.outputs, nil) }, "panic case %v failed", c)
	}
}

func TestGetPluginVersions(t *testing.T) {
	clearHub()
	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.1"}, nil, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.2"}, nil, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.3"}, nil, nil, nil, nil)
	versions := GetPluginVersions()
	assert.Equal(t, []string{"1.0.3", "1.0.2", "1.0.1", "1.0.0"}, versions)
}

func TestGetPluginDetail(t *testing.T) {
	clearHub()
	meta, err := GetPluginDetail("not exist version")
	assert.Nil(t, meta)
	assert.NotNil(t, err)

	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil, nil)
	meta, err = GetPluginDetail("1.0.0")
	assert.Nil(t, err)
	assert.NotNil(t, meta)
}

func TestGetPlugin(t *testing.T) {
	clearHub()
	plugin, err := GetPlugin("not exist version")
	assert.Nil(t, plugin)
	assert.NotNil(t, err)

	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil, nil)
	plugin, err = GetPlugin("1.0.0")
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.Equal(t, plugin.Version(), "1.0.0")
}
