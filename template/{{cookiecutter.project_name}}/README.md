# {{cookiecutter.project_name}}

{{cookiecutter.plugin_desc}}

## Structure

```text
.
├── app_desc.yml
├── bin/sync_apigateway.sh
├── go.mod
├── go.sum
├── main.go
└── versions/v100
    ├── forms/
    │   ├── form.js
    │   └── form.json
    ├── plugin.go
    └── plugin_test.go
```

The default version is `1.0.0`. It reads `hello` and writes `world` with the same value.

## Render Form (`forms.renderform`)

Each version can describe how the frontend renders the plugin inputs form. The
form files live under `versions/v100/forms/` and are embedded as a whole via
`//go:embed forms`. The framework resolves `data.forms.renderform` in the plugin
detail response with the following priority: **① an explicit `RenderForm`
provided by the runtime → ② `form.js` (passed through as-is) → ③ `null`**:

- `RenderForm` (runtime-provided, highest priority): set explicitly by the
  runtime and used directly as `forms.renderform`.
- `form.js` (JS): read from `forms/form.js` into `RenderFormJS` and passed to
  `hub.PluginSpec.RenderForm`. The content is passed through as-is (a raw
  string) without any parsing, mirroring the Python framework's `form.js`
  behavior.
- When neither is provided, `forms.renderform` is `null`.

`form.json` (read from `forms/form.json` into `InputsForm` and passed to
`hub.PluginSpec.Form`) is **never** exposed as `forms.renderform`. Instead, it
is **merged into the inputs schema** as the UI attributes of each field. When no
`form.js` is provided, `forms.renderform` stays `null` and the frontend falls
back to rendering the inputs schema (into which `form.json` has been merged).

Both files are optional and may be provided independently — you can keep only
`form.js`, only `form.json`, or both. However, at least one must exist: the
`//go:embed forms` directive requires the `forms/` directory to be non-empty,
otherwise the project fails to compile. A missing file is read as `nil` at
runtime and treated as "not provided" by the framework's `len()==0` fallback.

## Local Checks

The generated project already includes a tidy `go.mod` and matching `go.sum`, so the default project can be tested and built without resolving dependencies first.

```bash
go test -mod=readonly ./... -count=1
go build -mod=readonly ./...
```

After adding, removing, or upgrading dependencies, run `go mod tidy` and commit both `go.mod` and `go.sum`. If you override the template's default framework or runtime version during generation, run `go mod tidy` before the first deployment so the indirect dependency graph and checksums match the selected versions.

The default framework version is `{{cookiecutter.framework_version}}`. It must exist as an official Go module tag for normal dependency resolution. When validating an unreleased template branch locally, add a temporary `replace github.com/TencentBlueKing/bk-plugin-framework-go => <local-framework-checkout>` and remove it before release.

The template includes a public `replace github.com/TencentBlueKing/gopkg v1.3.0 => github.com/TencentBlueKing/gopkg v1.0.9` for `bk-plugin-runtime-go {{cookiecutter.runtime_version}}`. This matches the runtime repository and keeps `bk-apigateway-sdks v1.1.4` on the compatible cache API until a newer runtime tag removes the need.

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
