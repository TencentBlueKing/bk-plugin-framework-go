// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package hub implements a plugin hub. It provide the capability to:
//
// 1. install specific version to current bk-plugin project.
//
// 2. collect installed bk-plugin and it's meta data.
//
// 3. retrive information for specific version of bk-plugin.
package hub

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"

	"github.com/alecthomas/jsonschema"
)

// emptySchema will set to plugin when the inputs or outputs schema of this plugin is empty.
var emptySchema = []byte(`{"type": "object", "properties": {}, "required": [], "definitions": {}}`)

// versionre means the valid version code regex.
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9][a-z0-9]*$`)

// hub will store all installed plugin detail.
var hub = map[string]*PluginDetail{}

// clearHub will remove all version of current bk-plugin.
func clearHub() {
	hub = make(map[string]*PluginDetail)
}

// A PluginDetail store the detail data of specific plugin version.
type PluginDetail struct {
	plugin                  kit.Plugin
	contextInputsSchema     []byte
	outputsSchema           []byte
	inputsSchemaJSON        map[string]interface{}
	contextInputsSchemaJSON map[string]interface{}
	outputsSchemaJSON       map[string]interface{}
}

// Plugin returns the Plugin instance.
func (p *PluginDetail) Plugin() kit.Plugin {
	return p.plugin
}

// InputsSchema returns the plugin inputs json schema.

// ContextInputsSchema returns the plugin context inputs json schema.
func (p *PluginDetail) ContextInputsSchema() []byte {
	return p.contextInputsSchema
}

// OutputsSchema returns the plugin outputs json schema.
func (p *PluginDetail) OutputsSchema() []byte {
	return p.outputsSchema
}

// InputsSchemaJSON returns the unmarshaled plugin inputs json schema.
func (p *PluginDetail) InputsSchemaJSON() map[string]interface{} {
	return p.inputsSchemaJSON
}

// ContextInputsSchemaJSON returns the unmarshaled plugin context inputs json schema.
func (p *PluginDetail) ContextInputsSchemaJSON() map[string]interface{} {
	return p.contextInputsSchemaJSON
}

// OutputsSchemaJSON returns the unmarshaled plugin outputs json schema.
func (p *PluginDetail) OutputsSchemaJSON() map[string]interface{} {
	return p.outputsSchemaJSON
}

// reflectJSONSchema returns the byte array and string map of object's json schema.
func reflectJSONSchema(object interface{}, extraAttrs map[string]map[string]interface{}) ([]byte, map[string]interface{}, error) {
	if object == nil {
		var emptySchemaJSON map[string]interface{}
		if err := json.Unmarshal(emptySchema, &emptySchemaJSON); err != nil {
			return nil, nil, err
		}
		return emptySchema, emptySchemaJSON, nil
	}

	// generate json schema
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	objectSchema, err := reflector.Reflect(object).MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	// inject extraAttrs
	var objectSchemaJSON map[string]interface{}
	if err := json.Unmarshal(objectSchema, &objectSchemaJSON); err != nil {
		return nil, nil, err
	}

	properties := objectSchemaJSON["properties"].(map[string]interface{})
	if extraAttrs != nil {
		for prop := range extraAttrs {
			for k, v := range extraAttrs[prop] {
				if _, ok := properties[prop]; ok {
					property := properties[prop].(map[string]interface{})
					property[k] = v
				}
			}
		}

		// re-marshal schema
		objectSchema, err = json.Marshal(objectSchemaJSON)
		if err != nil {
			return nil, nil, err
		}
	}

	return objectSchema, objectSchemaJSON, nil
}

// MustInstall will install a version of plugin to hub.
//
// The p is the plugin will be installed.
//
// The inputs is inputs struct of this version, pass nil if this version
// do not have inputs.
//
// The contextInputs is context inputs struct of this version, pass nil if this version
// do not have context inputs.
//
// The outputs is outputs struct of this version, pass nil if this version
// do not have outputs.
//
// The inputsForm is json schema form for inputs
func MustInstall(p kit.Plugin, contextInputs interface{}, outputs interface{}, InputsForm []byte) {
	// versionw validation
	v := p.Version()
	if !versionRe.MatchString(v) {
		panic(fmt.Errorf("%s is not a valid plugin version\n", v))
	}

	if _, found := hub[v]; found {
		panic(fmt.Errorf("version %v already been installed\n", v))
	}

	// generate context inputs schema
	contextInputsSchema, contextInputsSchemaJSON, err := reflectJSONSchema(contextInputs, nil)
	if err != nil {
		panic(err)
	}

	// generate outputs schema
	outputsSchema, outputsSchemaJSON, err := reflectJSONSchema(outputs, nil)
	if err != nil {
		panic(err)
	}

	var inputsSchemaJSON = make(map[string]interface{})
	err = json.Unmarshal(InputsForm, &inputsSchemaJSON)
	if err != nil {
		panic(err)
	}

	hub[v] = &PluginDetail{
		plugin:                  p,
		contextInputsSchema:     contextInputsSchema,
		outputsSchema:           outputsSchema,
		inputsSchemaJSON:        inputsSchemaJSON,
		contextInputsSchemaJSON: contextInputsSchemaJSON,
		outputsSchemaJSON:       outputsSchemaJSON,
	}
}

// GetPluginVersions returns the versions of intalled plugin instance in new to old order.
func GetPluginVersions() []string {
	versions := make([]string, 0, len(hub))
	for k := range hub {
		versions = append(versions, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	return versions
}

// GetPluginDetail returns PluginDetail of specific plugin version.
func GetPluginDetail(v string) (*PluginDetail, error) {
	if meta, found := hub[v]; found {
		return meta, nil
	} else {
		return nil, fmt.Errorf("can not found plugin for version: %v", v)
	}
}

// GetPluginDetail returns Plugin of specific plugin version.
func GetPlugin(v string) (kit.Plugin, error) {
	meta, err := GetPluginDetail(v)
	if err != nil {
		return nil, err
	}
	return meta.plugin, nil
}
