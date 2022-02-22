package hub

import (
	"bpfgo/kit"
	"fmt"
	"regexp"
	"sort"

	"github.com/alecthomas/jsonschema"
)

var emptySchema = []byte(`{"type": "object", "properties": {}, "required": [], "definitions": {}}`)
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9][a-z0-9]*$`)
var hub = map[string]*PluginDetail{}

func clearHub() {
	hub = make(map[string]*PluginDetail)
}

type PluginDetail struct {
	plugin              kit.Plugin
	inputsSchema        []byte
	contextInputsSchema []byte
	outputsSchema       []byte
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

func reflectJSONSchema(inputs interface{}) ([]byte, error) {
	if inputs == nil {
		return emptySchema, nil
	}

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	return reflector.Reflect(inputs).MarshalJSON()
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
	inputsSchema, err := reflectJSONSchema(inputs)
	if err != nil {
		panic(err)
	}

	// generate context inputs schema
	contextInputsSchema, err := reflectJSONSchema(contextInputs)
	if err != nil {
		panic(err)
	}

	// generate outputs schema
	outputsSchema, err := reflectJSONSchema(outputs)
	if err != nil {
		panic(err)
	}

	hub[v] = &PluginDetail{plugin: p, inputsSchema: inputsSchema, contextInputsSchema: contextInputsSchema, outputsSchema: outputsSchema}
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
