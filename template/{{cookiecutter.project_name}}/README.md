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
    ├── form.json
    ├── plugin.go
    └── plugin_test.go
```

The default version is `1.0.0`. It reads `hello` and writes `world` with the same value.

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
