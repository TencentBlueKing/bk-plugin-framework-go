# Go Plugin Template Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Python-aligned Go plugin cookiecutter template and expose `MustInstallV2` form metadata through the Go detail protocol.

**Architecture:** Keep `bk-plugin-framework-go` as the SDK/protocol owner and make a narrow protocol change there. Add a minimal generated plugin project under `template/` that uses `bk-plugin-runtime-go` for server, worker, APIGW sync, and public-key fetch. The template defaults to framework `v1.0.4` as the first tag expected to contain the protocol fix, and runtime `v0.2.5`.

**Tech Stack:** Go 1.23 templates, `bk-plugin-framework-go`, `bk-plugin-runtime-go`, cookiecutter, PaaS v3 `app_desc.yml`, shell pre-release hook.

---

## File Map

- Modify `hub/registry.go`: track whether a plugin version has explicit renderform metadata.
- Modify `hub/registry_test.go`: lock legacy and `MustInstallV2` renderform flags.
- Modify `protocol/detail.go`: default detail `forms.renderform` from explicit `MustInstallV2` form metadata.
- Modify `protocol/protocol_test.go`: expect `MustInstallV2` renderform in detail and keep legacy renderform nil.
- Modify `info/version.go`: prepare framework version string `v1.0.4`.
- Create `template/cookiecutter.json`: user-facing template variables.
- Create `template/{{cookiecutter.project_name}}/.gitignore`: generated project ignores.
- Create `template/{{cookiecutter.project_name}}/README.md`: generated project usage and v2 deployment adaptation.
- Create `template/{{cookiecutter.project_name}}/app_desc.yml`: PaaS v3 deployment.
- Create `template/{{cookiecutter.project_name}}/go.mod`: generated module dependencies.
- Create `template/{{cookiecutter.project_name}}/main.go`: plugin registration and runtime entry.
- Create `template/{{cookiecutter.project_name}}/bin/sync_apigateway.sh`: pre-release APIGW sync.
- Create `template/{{cookiecutter.project_name}}/versions/v100/form.json`: default input form.
- Create `template/{{cookiecutter.project_name}}/versions/v100/plugin.go`: default hello/world plugin.
- Create `template/{{cookiecutter.project_name}}/versions/v100/plugin_test.go`: default plugin unit tests.

## Task 1: Align Detail Renderform Protocol

**Files:**
- Modify: `hub/registry.go`
- Modify: `hub/registry_test.go`
- Modify: `protocol/detail.go`
- Modify: `protocol/protocol_test.go`

- [ ] **Step 1: Write failing hub tests for explicit renderform tracking**

Update `TestPluginDetailPlugin`, `TestMustInstallLegacyKeepsInputsFormAsInputsSchema`, and `TestMustInstallV2StoresExplicitSchemasAndForm` in `hub/registry_test.go` with these assertions:

```go
func TestPluginDetailPlugin(t *testing.T) {
	plugin := MetaTestPlugin{}
	inputsSchema := []byte("{\"inputsSchema\": 1}")
	contextInputsSchema := []byte("{\"contextInputsSchema\": 2}")
	outputsSchema := []byte("{\"outputsSchema\": 3}")
	inputsSchemaJSON := map[string]interface{}{"inputsSchema": 1}
	contextInputsSchemaJSON := map[string]interface{}{"contextInputsSchema": 2}
	outputsSchemaJSON := map[string]interface{}{"outputsSchema": 3}
	formsRenderFormJSON := map[string]interface{}{"template_id": "render"}

	detail := PluginDetail{
		plugin:                  &plugin,
		inputsSchema:            inputsSchema,
		contextInputsSchema:     contextInputsSchema,
		outputsSchema:           outputsSchema,
		inputsSchemaJSON:        inputsSchemaJSON,
		contextInputsSchemaJSON: contextInputsSchemaJSON,
		outputsSchemaJSON:       outputsSchemaJSON,
		formsRenderFormJSON:     formsRenderFormJSON,
		formsRenderFormEnabled:  true,
	}

	assert.Equal(t, detail.Plugin(), &plugin)
	assert.Equal(t, detail.InputsSchema(), inputsSchema)
	assert.Equal(t, detail.ContextInputsSchema(), contextInputsSchema)
	assert.Equal(t, detail.OutputsSchema(), outputsSchema)
	assert.Equal(t, detail.InputsSchemaJSON(), inputsSchemaJSON)
	assert.Equal(t, detail.ContextInputsSchemaJSON(), contextInputsSchemaJSON)
	assert.Equal(t, detail.OutputsSchemaJSON(), outputsSchemaJSON)
	assert.Equal(t, detail.FormsRenderFormJSON(), formsRenderFormJSON)
	assert.True(t, detail.FormsRenderFormEnabled())
}
```

In `TestMustInstallLegacyKeepsInputsFormAsInputsSchema`, add:

```go
assert.False(t, detail.FormsRenderFormEnabled())
```

In `TestMustInstallV2StoresExplicitSchemasAndForm`, add:

```go
assert.True(t, detail.FormsRenderFormEnabled())
```

- [ ] **Step 2: Write failing protocol test for `MustInstallV2` renderform**

Replace `TestBuildDetailKeepsGoJSONSchemaOutOfRenderForm` in `protocol/protocol_test.go` with:

```go
func TestBuildDetailIncludesExplicitRenderForm(t *testing.T) {
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
	require.Equal(t, map[string]interface{}{
		"mode": map[string]interface{}{"component": "input"},
	}, data.Forms.RenderForm)

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"renderform":{"mode":{"component":"input"}}`)
}
```

Keep `TestBuildDetailKeepsLegacyInputsFormAsInputs` unchanged so it still requires legacy renderform to be nil.

- [ ] **Step 3: Run tests and confirm they fail for the intended reason**

Run:

```bash
go test ./hub ./protocol -count=1
```

Expected: FAIL because `FormsRenderFormEnabled` and `formsRenderFormEnabled` do not exist, and/or protocol detail still returns nil renderform.

- [ ] **Step 4: Implement explicit renderform tracking in `hub/registry.go`**

Add a field and accessor to `PluginDetail`:

```go
type PluginDetail struct {
	plugin                  kit.Plugin
	inputsSchema            []byte
	contextInputsSchema     []byte
	outputsSchema           []byte
	inputsSchemaJSON        map[string]interface{}
	contextInputsSchemaJSON map[string]interface{}
	outputsSchemaJSON       map[string]interface{}
	formsRenderFormJSON     map[string]interface{}
	formsRenderFormEnabled  bool
}
```

Add this method after `FormsRenderFormJSON()`:

```go
// FormsRenderFormEnabled returns whether the render form metadata should be
// exposed as forms.renderform in the plugin detail protocol.
func (p *PluginDetail) FormsRenderFormEnabled() bool {
	return p.formsRenderFormEnabled
}
```

In `mustInstallDetail`, compute the flag after form parsing:

```go
formsRenderFormEnabled := !legacyInputsFormAsSchema && len(spec.Form) > 0
if legacyInputsFormAsSchema {
	inputsSchema = spec.Form
	inputsSchemaJSON = formsRenderFormJSON
}
```

Store the flag in the hub entry:

```go
hub[v] = &PluginDetail{
	plugin:                  p,
	inputsSchema:            inputsSchema,
	contextInputsSchema:     contextInputsSchema,
	outputsSchema:           outputsSchema,
	inputsSchemaJSON:        inputsSchemaJSON,
	contextInputsSchemaJSON: contextInputsSchemaJSON,
	outputsSchemaJSON:       outputsSchemaJSON,
	formsRenderFormJSON:     formsRenderFormJSON,
	formsRenderFormEnabled:  formsRenderFormEnabled,
}
```

- [ ] **Step 5: Implement renderform defaulting in `protocol/detail.go`**

Replace `BuildDetail` with:

```go
// BuildDetail builds the standard plugin service detail payload.
func BuildDetail(version string, opts DetailOptions) (DetailData, error) {
	detail, err := hub.GetPluginDetail(version)
	if err != nil {
		return DetailData{}, err
	}

	renderForm := opts.RenderForm
	if renderForm == nil && detail.FormsRenderFormEnabled() {
		renderForm = detail.FormsRenderFormJSON()
	}

	return DetailData{
		Version:              detail.Plugin().Version(),
		Desc:                 detail.Plugin().Desc(),
		EnablePluginCallback: opts.EnablePluginCallback,
		Inputs:               detail.InputsSchemaJSON(),
		ContextInputs:        detail.ContextInputsSchemaJSON(),
		Outputs:              detail.OutputsSchemaJSON(),
		Forms: DetailForms{
			RenderForm: renderForm,
		},
	}, nil
}
```

- [ ] **Step 6: Run focused tests**

Run:

```bash
go test ./hub ./protocol -count=1
```

Expected: PASS.

- [ ] **Step 7: Run full framework tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit protocol alignment**

Run:

```bash
git add hub/registry.go hub/registry_test.go protocol/detail.go protocol/protocol_test.go
git commit -m "feat: expose explicit plugin render form"
```

## Task 2: Prepare Framework Version for Template Pin

**Files:**
- Modify: `info/version.go`
- Test: `info/version_test.go`

- [ ] **Step 1: Update framework version string**

Change `info/version.go` to:

```go
// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package info store basic information of bk-plugin-framework-go.
package info

const (
	version = "v1.0.4"
)

// Version returns the current version number of bk-plugin-framework-go.
func Version() string {
	return version
}
```

- [ ] **Step 2: Run version tests**

Run:

```bash
go test ./info -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit version prep**

Run:

```bash
git add info/version.go
git commit -m "chore: bump framework version to v1.0.4"
```

## Task 3: Add Template Project Skeleton

**Files:**
- Create: `template/cookiecutter.json`
- Create: `template/{{cookiecutter.project_name}}/.gitignore`
- Create: `template/{{cookiecutter.project_name}}/README.md`
- Create: `template/{{cookiecutter.project_name}}/app_desc.yml`
- Create: `template/{{cookiecutter.project_name}}/go.mod`
- Create: `template/{{cookiecutter.project_name}}/bin/sync_apigateway.sh`

- [ ] **Step 1: Create `template/cookiecutter.json`**

Create the file with:

```json
{
  "project_name": "my_plugin",
  "app_code": "the app code of plugin saas",
  "plugin_desc": "the description of this plugin",
  "init_apigw_maintainer": "admin",
  "framework_version": "v1.0.4",
  "runtime_version": "v0.2.5"
}
```

- [ ] **Step 2: Create generated project `.gitignore`**

Create `template/{{cookiecutter.project_name}}/.gitignore` with:

```gitignore
# Build outputs
bin/apigw.pub
dist/
build/
*.test

# Go
vendor/
coverage.out

# Logs and local env
v3logs/
.env

# Editors and OS files
.idea/
.vscode/
.DS_Store
```

- [ ] **Step 3: Create generated project `README.md`**

Create `template/{{cookiecutter.project_name}}/README.md` with:

```markdown
# {{cookiecutter.project_name}}

{{cookiecutter.plugin_desc}}

## Structure

```text
.
├── app_desc.yml
├── bin/sync_apigateway.sh
├── main.go
└── versions/v100
    ├── form.json
    ├── plugin.go
    └── plugin_test.go
```

The default version is `1.0.0`. It reads `hello` and writes `world` with the same value.

## Local Checks

```bash
go test ./... -count=1
go build ./... 
```

## Runtime Commands

```bash
{{cookiecutter.project_name}} server
{{cookiecutter.project_name}} worker
{{cookiecutter.project_name}} syncapigw
{{cookiecutter.project_name}} fetch-apigw-public-key
```

## PaaS Deployment

`app_desc.yml` uses PaaS v3 by default. The pre-release hook runs:

```bash
bash bin/sync_apigateway.sh
```

The script syncs the runtime-owned APIGW resources and fetches the gateway public key into `bin/apigw.pub`.

Important environment variables are injected by PaaS or configured on the app:

- `BKPAAS_APP_ID`
- `BKPAAS_APP_SECRET`
- `BKPAAS_DEFAULT_PREALLOCATED_URLS`
- `BK_APIGW_MANAGER_URL_TMPL`
- `BK_APIGW_MAINTAINERS`

The default synchronous plugin does not require callback configuration. If you enable callback features later, configure `BK_PLUGIN_CALLBACK_TOKEN_SECRET`.

## Legacy `spec_version: 2` Deployment

If the target environment still uses the older Go app descriptor shape, adapt the process section to:

```yaml
spec_version: 2
modules:
  default:
    language: go
    is_default: true
    processes:
      web:
        command: {{cookiecutter.project_name}} server
      worker:
        command: {{cookiecutter.project_name}} worker
    services:
      - name: mysql
      - name: redis
```

Add RabbitMQ only if your platform requires it for this runtime deployment.
```

- [ ] **Step 4: Create PaaS v3 `app_desc.yml`**

Create `template/{{cookiecutter.project_name}}/app_desc.yml` with:

```yaml
specVersion: 3
modules:
  - name: default
    isDefault: true
    language: go
    spec:
      hooks:
        preRelease:
          procCommand: bash bin/sync_apigateway.sh
      processes:
        - name: web
          procCommand: {{cookiecutter.project_name}} server
          services:
            - name: web
              exposedType:
                name: bk/http
              targetPort: 5000
              port: 80
        - name: worker
          procCommand: {{cookiecutter.project_name}} worker
      services:
        - name: mysql
        - name: redis
      configuration:
        env:
          - name: BK_APIGW_MAINTAINERS
            value: {{cookiecutter.init_apigw_maintainer}}
            description: plugin apigw maintainers
```

- [ ] **Step 5: Create generated project `go.mod`**

Create `template/{{cookiecutter.project_name}}/go.mod` with:

```go
// +heroku install {{cookiecutter.project_name}}
// +heroku goVersion go1.23
module {{cookiecutter.project_name}}

go 1.23.0

require (
	github.com/TencentBlueKing/bk-plugin-framework-go {{cookiecutter.framework_version}}
	github.com/TencentBlueKing/bk-plugin-runtime-go {{cookiecutter.runtime_version}}
	github.com/sirupsen/logrus v1.9.2
)
```

- [ ] **Step 6: Create APIGW sync script**

Create `template/{{cookiecutter.project_name}}/bin/sync_apigateway.sh` with:

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "[Sync] BEGIN ====================="

{{cookiecutter.project_name}} syncapigw
{{cookiecutter.project_name}} fetch-apigw-public-key

echo "[Sync] DONE ====================="
```

- [ ] **Step 7: Commit template skeleton**

Run:

```bash
git add template/cookiecutter.json \
  'template/{{cookiecutter.project_name}}/.gitignore' \
  'template/{{cookiecutter.project_name}}/README.md' \
  'template/{{cookiecutter.project_name}}/app_desc.yml' \
  'template/{{cookiecutter.project_name}}/go.mod' \
  'template/{{cookiecutter.project_name}}/bin/sync_apigateway.sh'
git commit -m "feat: add go plugin template skeleton"
```

## Task 4: Add Default Plugin Version and Tests

**Files:**
- Create: `template/{{cookiecutter.project_name}}/main.go`
- Create: `template/{{cookiecutter.project_name}}/versions/v100/form.json`
- Create: `template/{{cookiecutter.project_name}}/versions/v100/plugin.go`
- Create: `template/{{cookiecutter.project_name}}/versions/v100/plugin_test.go`

- [ ] **Step 1: Create generated project `main.go`**

Create `template/{{cookiecutter.project_name}}/main.go` with:

```go
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
```

- [ ] **Step 2: Create `form.json`**

Create `template/{{cookiecutter.project_name}}/versions/v100/form.json` with:

```json
{
  "hello": {
    "type": "string",
    "title": "Hello",
    "default": "",
    "ui:component": {
      "name": "bk-input",
      "props": {}
    },
    "ui:rules": [
      "required"
    ]
  }
}
```

- [ ] **Step 3: Create default plugin implementation**

Create `template/{{cookiecutter.project_name}}/versions/v100/plugin.go` with:

```go
package v100

import (
	_ "embed"
	"fmt"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
)

//go:embed form.json
var InputsForm []byte

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
```

- [ ] **Step 4: Create default plugin tests**

Create `template/{{cookiecutter.project_name}}/versions/v100/plugin_test.go` with:

```go
package v100

import (
	"encoding/json"
	"testing"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/sirupsen/logrus"
)

type testReader struct {
	inputs        map[string]interface{}
	contextInputs map[string]interface{}
}

func (r testReader) ReadInputs(v interface{}) error {
	return marshalTo(r.inputs, v)
}

func (r testReader) ReadContextInputs(v interface{}) error {
	return marshalTo(r.contextInputs, v)
}

type testStore struct {
	data map[string][]byte
}

func newTestStore() *testStore {
	return &testStore{data: map[string][]byte{}}
}

func (s *testStore) Write(traceID string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.data[traceID] = data
	return nil
}

func (s *testStore) Read(traceID string, v interface{}) error {
	return json.Unmarshal(s.data[traceID], v)
}

func marshalTo(src interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func TestHelloWorldPluginWritesOutput(t *testing.T) {
	contextStore := newTestStore()
	outputsStore := newTestStore()
	reader := testReader{
		inputs: map[string]interface{}{
			"hello": "world",
		},
		contextInputs: map[string]interface{}{
			"executor": "admin",
		},
	}

	plugin := &Plugin{}
	c := kit.NewContext("trace-hello", constants.StateEmpty, 1, reader, contextStore, outputsStore, logrus.NewEntry(logrus.StandardLogger()))
	if err := plugin.Execute(c); err != nil {
		t.Fatalf("execute plugin: %v", err)
	}
	if c.WaitingPoll() || c.WaitingCallback() {
		t.Fatalf("default plugin should finish synchronously")
	}

	var outputs Outputs
	if err := outputsStore.Read("trace-hello", &outputs); err != nil {
		t.Fatalf("read outputs: %v", err)
	}
	if outputs.World != "world" {
		t.Fatalf("World = %q, want %q", outputs.World, "world")
	}
}

func TestHelloWorldPluginRejectsUnsupportedState(t *testing.T) {
	contextStore := newTestStore()
	outputsStore := newTestStore()
	reader := testReader{
		inputs:        map[string]interface{}{"hello": "world"},
		contextInputs: map[string]interface{}{"executor": "admin"},
	}

	plugin := &Plugin{}
	c := kit.NewContext("trace-poll", constants.StatePoll, 2, reader, contextStore, outputsStore, logrus.NewEntry(logrus.StandardLogger()))
	if err := plugin.Execute(c); err == nil {
		t.Fatalf("expected unsupported state error")
	}
}
```

- [ ] **Step 5: Commit default plugin**

Run:

```bash
git add 'template/{{cookiecutter.project_name}}/main.go' \
  'template/{{cookiecutter.project_name}}/versions/v100/form.json' \
  'template/{{cookiecutter.project_name}}/versions/v100/plugin.go' \
  'template/{{cookiecutter.project_name}}/versions/v100/plugin_test.go'
git commit -m "feat: add default go plugin version"
```

## Task 5: Render Template and Validate Generated Project

**Files:**
- Generated temporary files under `/tmp`, not committed.
- Template `go.sum` handling depends on release availability.

- [ ] **Step 1: Render the template with cookiecutter**

Run:

```bash
rm -rf /tmp/bk-plugin-template-render
mkdir -p /tmp/bk-plugin-template-render
uvx cookiecutter==2.6.0 --no-input template \
  --output-dir /tmp/bk-plugin-template-render \
  project_name=rendered_go_plugin \
  app_code=rendered_go_plugin \
  plugin_desc="Rendered Go plugin" \
  init_apigw_maintainer=admin \
  framework_version=v1.0.4 \
  runtime_version=v0.2.5
```

Expected: `/tmp/bk-plugin-template-render/rendered_go_plugin` exists.

- [ ] **Step 2: Add a temporary local replace for pre-release validation**

Until `github.com/TencentBlueKing/bk-plugin-framework-go v1.0.4` is published, run:

```bash
cd /tmp/bk-plugin-template-render/rendered_go_plugin
go mod edit -replace github.com/TencentBlueKing/bk-plugin-framework-go=/Users/dengyh/Projects/bk-plugin-framework-go
```

Expected: `go.mod` contains a local replace. This replace is for validation only and must not be copied back into `template/`.

- [ ] **Step 3: Tidy generated project**

Run:

```bash
cd /tmp/bk-plugin-template-render/rendered_go_plugin
go mod tidy
```

Expected: PASS and a generated `go.sum` appears in the temporary rendered project.

- [ ] **Step 4: Run generated project tests**

Run:

```bash
cd /tmp/bk-plugin-template-render/rendered_go_plugin
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Build generated project**

Run:

```bash
cd /tmp/bk-plugin-template-render/rendered_go_plugin
go build ./...
```

Expected: PASS.

- [ ] **Step 6: Verify rendered files contain expected runtime wiring**

Run:

```bash
rg -n "bk-plugin-runtime-go|MustInstallV2|syncapigw|fetch-apigw-public-key|specVersion: 3" /tmp/bk-plugin-template-render/rendered_go_plugin
```

Expected: output includes `go.mod`, `main.go`, `bin/sync_apigateway.sh`, and `app_desc.yml`.

- [ ] **Step 7: Decide template `go.sum` inclusion**

If `v1.0.4` has been published before implementation reaches this step, remove the temporary replace, run `go mod tidy`, and copy the generated `go.sum` into `template/{{cookiecutter.project_name}}/go.sum`.

If `v1.0.4` has not been published, do not add `template/{{cookiecutter.project_name}}/go.sum`; instead, keep `go.mod` only and document in the final implementation report that official-tag validation is blocked until the `v1.0.4` tag is published. The generated project still validates with the local replace above.

- [ ] **Step 8: Commit template validation adjustments**

If `template/{{cookiecutter.project_name}}/go.sum` was added, run:

```bash
git add 'template/{{cookiecutter.project_name}}/go.sum'
git commit -m "chore: add go plugin template sums"
```

If no `go.sum` was added because `v1.0.4` is unpublished, skip this commit and record the reason in the final implementation report.

## Task 6: Static Checks and Final Documentation Consistency

**Files:**
- Modify only files that fail the checks from earlier tasks.

- [ ] **Step 1: Check template has no old runtime remnants**

Run:

```bash
rg -n "beego-runtime|app.json|post-compile|database/migrations" template || true
```

Expected: no output.

- [ ] **Step 2: Check dependency pins**

Run:

```bash
rg -n '"framework_version": "v1.0.4"|"runtime_version": "v0.2.5"|github.com/TencentBlueKing/bk-plugin-framework-go {{cookiecutter.framework_version}}|github.com/TencentBlueKing/bk-plugin-runtime-go {{cookiecutter.runtime_version}}' template
```

Expected: output includes `template/cookiecutter.json` and `template/{{cookiecutter.project_name}}/go.mod`.

- [ ] **Step 3: Run full framework tests again**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run diff whitespace check**

Run:

```bash
git diff --check
```

Expected: no output.

- [ ] **Step 5: Review final status**

Run:

```bash
git status --short --branch
git log --oneline --decorate --max-count=8
```

Expected: branch includes the implementation commits from Tasks 1-4, and only pre-existing unrelated untracked files such as `.serena/` and `docs/session-handoff/` remain untracked.
