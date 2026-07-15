package template_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type templateConfig struct {
	FrameworkVersion string `json:"framework_version"`
	RuntimeVersion   string `json:"runtime_version"`
}

func TestDefaultDependencyFilesAreComplete(t *testing.T) {
	configData, err := os.ReadFile("cookiecutter.json")
	if err != nil {
		t.Fatalf("read cookiecutter config: %v", err)
	}

	var config templateConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("decode cookiecutter config: %v", err)
	}

	goModData, err := os.ReadFile("{{cookiecutter.project_name}}/go.mod")
	if err != nil {
		t.Fatalf("read template go.mod: %v", err)
	}
	goMod := string(goModData)
	if !strings.Contains(goMod, "// indirect") {
		t.Fatal("template go.mod must include the tidy indirect dependency graph")
	}

	goSumData, err := os.ReadFile("{{cookiecutter.project_name}}/go.sum")
	if err != nil {
		t.Fatalf("read template go.sum: %v", err)
	}
	goSum := string(goSumData)

	dependencies := []struct {
		module  string
		version string
	}{
		{"github.com/TencentBlueKing/bk-plugin-framework-go", config.FrameworkVersion},
		{"github.com/TencentBlueKing/bk-plugin-runtime-go", config.RuntimeVersion},
	}
	for _, dependency := range dependencies {
		for _, suffix := range []string{" h1:", "/go.mod h1:"} {
			entry := dependency.module + " " + dependency.version + suffix
			if !strings.Contains(goSum, entry) {
				t.Errorf("template go.sum missing %q", entry)
			}
		}
	}
}
