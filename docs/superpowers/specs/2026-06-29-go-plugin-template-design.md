# Go Plugin Template Design

## Context

This work adds a cookiecutter template for new Go BK standard plugins and aligns the Go detail protocol with the Python plugin framework where the template depends on it.

The current `origin/main_with_template` branch is useful as an early Go template reference, but it targets the older `beego-runtime`, uses legacy `hub.MustInstall`, includes a Beego migration stub, and keeps a Heroku-style `app.json`. The new template targets the current split architecture:

- `bk-plugin-framework-go v1.0.3` as the plugin SDK.
- `bk-plugin-runtime-go v0.2.5` as the HTTP server, worker, APIGW sync, callback, and finish-callback runtime.

Those are the latest observed released tags during design. If the protocol alignment fix below requires a newer framework or runtime release, the template defaults must be updated to the first official tags that contain the fix before claiming that generated projects expose `forms.renderform`.

The Python template is the user-experience reference: after cookiecutter initialization, users should be able to deploy and run a default plugin without editing business code.

## Goals

- Add `template/` for initializing a new Go plugin project.
- Default to a minimal synchronous `hello -> world` plugin matching the Python template's simplicity.
- Use official Go module tags, not vendored `third_party` runtime code.
- Default deployment to PaaS v3 `app_desc.yml`, while documenting how to adapt to legacy `spec_version: 2` deployments.
- Run APIGW sync and public-key fetch through a pre-release script.
- Make the detail response expose `forms.renderform` for `hub.MustInstallV2` plugins so the template's form metadata is not silently dropped.
- Keep legacy `hub.MustInstall` behavior compatible.

## Non-Goals

- Do not vendor `bk-plugin-runtime-go` into generated projects.
- Do not add poll, callback, or finish-callback examples to the default generated plugin.
- Do not migrate existing Go plugin projects.
- Do not replace the runtime's embedded APIGW `definition.yaml` and `resources.yaml` defaults.
- Do not remove legacy `hub.MustInstall`.

## Recommended Approach

Use a small template plus a small protocol fix.

The template stays clean and deployable. Runtime/framework protocol behavior is corrected only where it affects Python alignment and template correctness: `forms.renderform` should return the form registered through `MustInstallV2`.

## Template Structure

```text
template/
  cookiecutter.json
  {{cookiecutter.project_name}}/
    .gitignore
    README.md
    app_desc.yml
    go.mod
    go.sum
    main.go
    bin/
      sync_apigateway.sh
    versions/
      v100/
        form.json
        plugin.go
        plugin_test.go
```

Files deliberately omitted from `main_with_template`:

- `app.json`: Heroku example metadata, not needed for BK PaaS.
- `bin/pre-compile`: no template-specific compile flags are required.
- `bin/post-compile`: APIGW sync belongs in the v3 pre-release hook.
- `database/migrations/...`: the new runtime owns its schedule-store setup; the plugin template should not contain a Beego migration stub.

## Cookiecutter Variables

`template/cookiecutter.json` should include:

- `project_name`: default `my_plugin`.
- `app_code`: the plugin SaaS app code.
- `plugin_desc`: default plugin description.
- `init_apigw_maintainer`: default `admin`.
- `framework_version`: default to the latest official framework tag that includes the detail/form alignment behavior.
- `runtime_version`: default to the latest official runtime tag compatible with that framework tag.

If no released tag contains the required detail/form behavior yet, the implementation plan must include the release/version-bump step or leave the defaults configurable without claiming release-tag end-to-end success.

APIGW manager URL, timeout, release version, callback secret, and backend host should not be hardcoded in the template. They are runtime/PaaS environment concerns and already have documented conventions in `bk-plugin-runtime-go`.

## Generated Plugin Behavior

The generated plugin has one version: `1.0.0` in `versions/v100`.

Types:

- `Inputs`: `{ hello string }`.
- `ContextInputs`: `{ executor string }`.
- `Outputs`: `{ world string }`.

Execution:

1. If the context state is not `constants.StateEmpty`, return a clear unsupported-state error.
2. Read `Inputs`.
3. Read `ContextInputs`.
4. Write `Outputs{World: inputs.Hello}`.
5. Return nil, causing a synchronous success.

The default plugin does not call `WaitPoll`, `PrepareCallback`, or `WaitCallback`.

## Registration

`main.go` should register the version with `hub.MustInstallV2`:

```go
hub.MustInstallV2(&v100.Plugin{}, hub.PluginSpec{
    Inputs:        v100.Inputs{},
    ContextInputs: v100.ContextInputs{},
    Outputs:       v100.Outputs{},
    Form:          v100.InputsForm,
})
```

Then it should call `runner.Run()` from `github.com/TencentBlueKing/bk-plugin-runtime-go/runner`.

Finish callback is not enabled by default. The README can show users how to opt in with `hub.Configure(hub.Options{EnablePluginCallback: true})`.

## Deployment Flow

`app_desc.yml` should default to PaaS v3 using the Python template's shape:

- `specVersion: 3`.
- `modules` list with module `default`.
- `spec.hooks.preRelease.procCommand: bash bin/sync_apigateway.sh`.
- `spec.processes` containing:
  - web process: `{{cookiecutter.project_name}} server`.
  - worker process: `{{cookiecutter.project_name}} worker`.
- services: MySQL and Redis.
- env values:
  - `BK_APIGW_MAINTAINERS={{cookiecutter.init_apigw_maintainer}}`.

`bin/sync_apigateway.sh` should:

```bash
{{cookiecutter.project_name}} syncapigw
{{cookiecutter.project_name}} fetch-apigw-public-key
```

The script should use `set -euo pipefail` and print concise begin/done logs.

The generated README should include a short `spec_version: 2` adaptation snippet for older Go plugin deployment environments:

- `processes.web.command: {{cookiecutter.project_name}} server`
- `processes.worker.command: {{cookiecutter.project_name}} worker`
- services: MySQL and Redis, with RabbitMQ added only if the target platform requires it.

## Protocol Alignment

Python framework detail responses return:

- generated input schema under `inputs`;
- generated context schema under `context_inputs`;
- generated output schema under `outputs`;
- render form content under `forms.renderform`.

Current Go framework stores `PluginDetail.FormsRenderFormJSON()` for `MustInstallV2`, but `protocol.BuildDetail` only returns `DetailOptions.RenderForm`. Current runtime calls `BuildDetail` without passing the stored form. This means `forms.renderform` can be null even when a plugin registered form metadata.

The implementation should make `MustInstallV2` detail responses include the stored form as `forms.renderform`. Legacy `hub.MustInstall` should keep its existing behavior: the legacy form remains the input schema compatibility payload and should not unexpectedly appear as a separate renderform.

Acceptable implementation shape:

- Update `protocol.BuildDetail` to default `forms.renderform` from the plugin detail's stored form metadata.
- Preserve the ability for runtime options to override renderform if needed.
- Adjust runtime tests that currently assert `renderform == nil` for `MustInstallV2`.
- Keep tests that assert legacy `MustInstall` does not expose a separate renderform.

## Error Handling

Template plugin errors should be explicit and ordinary Go errors:

- unsupported state: include the numeric state value;
- input/context read failures: return the original read error;
- output write failure: return the original write error.

The sync script should fail fast. Any `syncapigw` or `fetch-apigw-public-key` failure should fail the pre-release hook, matching the Python template's deployment posture.

The generated README should call out that callback features require `BK_PLUGIN_CALLBACK_TOKEN_SECRET`, but the default synchronous plugin does not require callback configuration.

## Testing

Framework/runtime protocol tests:

- `go test ./protocol -count=1`
- `go test ./hub -count=1`
- runtime-side tests that cover `/bk_plugin/detail/:version` after the protocol fix, if the runtime repository is part of the implementation pass.

Template tests:

- Render the cookiecutter template into a temporary project.
- Run `go test ./... -count=1` in the rendered project.
- Verify `go test` exercises the default plugin with an in-memory `kit.Context`.
- Optionally run `go build ./...` in the rendered project.

Static checks:

- `git diff --check`
- Ensure `rg "beego-runtime|app.json|post-compile|database/migrations" template` has no unexpected matches.
- Ensure the template dependency pins point at official tags that include the protocol behavior required by this design.

## Success Criteria

- A newly rendered Go plugin project compiles and passes tests without user code changes.
- The default plugin can be deployed with PaaS v3 configuration.
- APIGW sync and public-key fetch are wired through the generated pre-release hook.
- Detail API for the default plugin includes explicit schemas and `forms.renderform`.
- Legacy `MustInstall` compatibility remains intact.
- The template is smaller and cleaner than `origin/main_with_template`, while keeping the useful cookiecutter shape.
