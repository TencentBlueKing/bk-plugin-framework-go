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
