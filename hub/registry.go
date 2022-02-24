package hub

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/homholueng/bk-plugin-framework-go/kit"

	"github.com/alecthomas/jsonschema"
)

var emptySchema = []byte(`{"type": "object", "properties": {}, "required": [], "definitions": {}}`)
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9][a-z0-9]*$`)
var hub = map[string]*PluginDetail{}

func clearHub() {
	hub = make(map[string]*PluginDetail)
}

type PluginDetail struct {
	plugin                  kit.Plugin
	inputsSchema            []byte
	contextInputsSchema     []byte
	outputsSchema           []byte
	inputsSchemaJSON        map[string]interface{}
	contextInputsSchemaJSON map[string]interface{}
	outputsSchemaJSON       map[string]interface{}
}

func (p *PluginDetail) Plugin() kit.Plugin {
	return p.plugin
}

func (p *PluginDetail) InputsSchema() []byte {
	return p.inputsSchema
}

func (p *PluginDetail) ContextInputsSchema() []byte {
	return p.contextInputsSchema
}

func (p *PluginDetail) OutputsSchema() []byte {
	return p.outputsSchema
}

func (p *PluginDetail) InputsSchemaJSON() map[string]interface{} {
	return p.inputsSchemaJSON
}

func (p *PluginDetail) ContextInputsSchemaJSON() map[string]interface{} {
	return p.contextInputsSchemaJSON
}

func (p *PluginDetail) OutputsSchemaJSON() map[string]interface{} {
	return p.outputsSchemaJSON
}

func reflectJSONSchema(object interface{}) ([]byte, map[string]interface{}, error) {
	if object == nil {
		var emptySchemaJSON map[string]interface{}
		if err := json.Unmarshal(emptySchema, &emptySchemaJSON); err != nil {
			return nil, nil, err
		}
		return emptySchema, emptySchemaJSON, nil
	}

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	objectSchema, err := reflector.Reflect(object).MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	var objectSchemaJSON map[string]interface{}
	if err := json.Unmarshal(objectSchema, &objectSchemaJSON); err != nil {
		return nil, nil, err
	}
	return objectSchema, objectSchemaJSON, nil
}

func MustInstall(p kit.Plugin, inputs interface{}, contextInputs interface{}, outputs interface{}) {
	// versionw validation
	v := p.Version()
	if !versionRe.MatchString(v) {
		panic(fmt.Errorf("%s is not a valid plugin version\n", v))
	}

	if _, found := hub[v]; found {
		panic(fmt.Errorf("version %v already been installed\n", v))
	}

	// generate inputs schema
	inputsSchema, inputsSchemaJSON, err := reflectJSONSchema(inputs)
	if err != nil {
		panic(err)
	}

	// generate context inputs schema
	contextInputsSchema, contextInputsSchemaJSON, err := reflectJSONSchema(contextInputs)
	if err != nil {
		panic(err)
	}

	// generate outputs schema
	outputsSchema, outputsSchemaJSON, err := reflectJSONSchema(outputs)
	if err != nil {
		panic(err)
	}

	hub[v] = &PluginDetail{
		plugin:                  p,
		inputsSchema:            inputsSchema,
		contextInputsSchema:     contextInputsSchema,
		outputsSchema:           outputsSchema,
		inputsSchemaJSON:        inputsSchemaJSON,
		contextInputsSchemaJSON: contextInputsSchemaJSON,
		outputsSchemaJSON:       outputsSchemaJSON,
	}
}

func GetPluginVersions() []string {
	versions := make([]string, 0, len(hub))
	for k := range hub {
		versions = append(versions, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	return versions
}

func GetPluginDetail(v string) (*PluginDetail, error) {
	if meta, found := hub[v]; found {
		return meta, nil
	} else {
		return nil, fmt.Errorf("can not found plugin for version: %v", v)
	}
}

func GetPlugin(v string) (kit.Plugin, error) {
	meta, err := GetPluginDetail(v)
	if err != nil {
		return nil, err
	}
	return meta.plugin, nil
}
