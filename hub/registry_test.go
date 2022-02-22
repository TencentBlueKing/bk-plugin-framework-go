package hub

import (
	"bpfgo/kit"
	"testing"

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
	inputSchema := []byte("inputSchema")
	contextInputSchema := []byte("contextInputSchema")
	outputSchema := []byte("outputSchema")

	meta := PluginDetail{
		plugin:              &plugin,
		inputsSchema:        inputSchema,
		contextInputsSchema: contextInputSchema,
		outputsSchema:       outputSchema,
	}

	assert.Equal(t, meta.Plugin(), &plugin)
	assert.Equal(t, meta.InputsSchema(), inputSchema)
	assert.Equal(t, meta.ContextInputsSchema(), contextInputSchema)
	assert.Equal(t, meta.OutputsSchema(), outputSchema)
}

func TestReflectJSONSchema(t *testing.T) {
	type ReflectStruct struct {
		TemplateID int    `json:"template_id"`
		TaskName   string `json:"task_name"`
	}
	var rs ReflectStruct
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema, err := reflector.Reflect(&rs).MarshalJSON()
	assert.Nil(t, err)

	var cases = []struct {
		in       interface{}
		expected []byte
	}{
		{rs, schema},
		{nil, emptySchema},
	}

	for _, c := range cases {
		actual, err := reflectJSONSchema(c.in)
		assert.Nil(t, err)
		assert.Equal(t, actual, c.expected)
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

func TestMustInstall(t *testing.T) {
	clearHub()
	var success_cases = []struct {
		plugin        *MustInstallTestPlugin
		inputs        interface{}
		contextInputs interface{}
		outputs       interface{}
	}{
		{&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil},
		{&MustInstallTestPlugin{version: "1.0.1"}, MustInstallTestPluginInput{}, nil, nil},
		{&MustInstallTestPlugin{version: "1.0.2"}, nil, MustInstallTestPluginContextInput{}, nil},
		{&MustInstallTestPlugin{version: "1.0.3"}, nil, nil, MustInstallTestPluginOutput{}},
		{&MustInstallTestPlugin{version: "1.0.4"}, MustInstallTestPluginInput{}, MustInstallTestPluginContextInput{}, MustInstallTestPluginOutput{}},
	}

	for _, c := range success_cases {
		assert.NotPanics(t, func() { MustInstall(c.plugin, c.inputs, c.contextInputs, c.outputs) }, "success case %v failed", c)
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
		assert.Panics(t, func() { MustInstall(c.plugin, c.inputs, c.contextInputs, c.outputs) }, "panic case %v failed", c)
	}
}

func TestGetPluginVersions(t *testing.T) {
	clearHub()
	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.1"}, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.2"}, nil, nil, nil)
	MustInstall(&MustInstallTestPlugin{version: "1.0.3"}, nil, nil, nil)
	versions := GetPluginVersions()
	assert.Equal(t, []string{"1.0.3", "1.0.2", "1.0.1", "1.0.0"}, versions)
}

func TestGetPluginDetail(t *testing.T) {
	clearHub()
	meta, err := GetPluginDetail("not exist version")
	assert.Nil(t, meta)
	assert.NotNil(t, err)

	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil)
	meta, err = GetPluginDetail("1.0.0")
	assert.Nil(t, err)
	assert.NotNil(t, meta)
}

func TestGetPlugin(t *testing.T) {
	clearHub()
	plugin, err := GetPlugin("not exist version")
	assert.Nil(t, plugin)
	assert.NotNil(t, err)

	MustInstall(&MustInstallTestPlugin{version: "1.0.0"}, nil, nil, nil)
	plugin, err = GetPlugin("1.0.0")
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.Equal(t, plugin.Version(), "1.0.0")
}
