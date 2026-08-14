package main

import (
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/runner"
	v100 "{{cookiecutter.project_name}}/versions/v100"
)

func main() {
	hub.MustInstallV2(&v100.Plugin{}, hub.PluginSpec{
		Inputs:        v100.Inputs{},
		ContextInputs: v100.ContextInputs{},
		Outputs:       v100.Outputs{},
		Form:          v100.InputsForm,
	})
	runner.Run()
}
