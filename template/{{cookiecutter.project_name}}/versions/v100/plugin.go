package v100

import (
	"embed"
	"fmt"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
)

//go:embed forms
var formFS embed.FS

// readForm reads an optional form file under forms/. It returns nil when the
// file is absent, so form.json and form.js can be provided independently.
// At least one of them must exist, otherwise the embed above fails to compile.
func readForm(name string) []byte {
	data, err := formFS.ReadFile("forms/" + name)
	if err != nil {
		return nil
	}
	return data
}

var (
	// InputsForm carries form.json (UI attributes merged into inputs schema).
	InputsForm = readForm("form.json")
	// RenderFormJS carries form.js (raw renderform, takes precedence).
	RenderFormJS = readForm("form.js")
)

// Inputs defines the visible plugin inputs.
type Inputs struct {
	Hello string `json:"hello" jsonschema:"title=Hello"`
}

// ContextInputs defines Standard Ops context inputs used by the plugin.
type ContextInputs struct {
	Executor string `json:"executor" jsonschema:"title=Executor"`
}

// Outputs defines the values returned to the caller.
type Outputs struct {
	World string `json:"world" jsonschema:"title=World"`
}

// Plugin implements version 1.0.0.
type Plugin struct{}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Desc returns the plugin description.
func (p *Plugin) Desc() string {
	return "{{cookiecutter.plugin_desc}}"
}

// Execute runs the synchronous hello/world plugin.
func (p *Plugin) Execute(c *kit.Context) error {
	if c.State() != constants.StateEmpty {
		return fmt.Errorf("hello world plugin does not support state %v", c.State())
	}

	var inputs Inputs
	if err := c.ReadInputs(&inputs); err != nil {
		return err
	}

	var contextInputs ContextInputs
	if err := c.ReadContextInputs(&contextInputs); err != nil {
		return err
	}
	_ = contextInputs

	return c.WriteOutputs(&Outputs{World: inputs.Hello})
}
