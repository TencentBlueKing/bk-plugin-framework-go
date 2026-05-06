# Blueapps Runtime Refactor Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first compatible Go plugin runtime that replaces `beego-runtime` for sync and poll plugins while keeping existing plugin business code stable.

**Architecture:** Keep `bk-plugin-framework-go` as the lightweight SDK and create a sibling `bk-plugin-runtime-go` module for the blueapps-go based runtime. Phase 1 implements SDK registration compatibility, runtime command entrypoints, meta/detail/invoke/schedule HTTP protocol, durable DB-backed poll scheduling, and migration documentation.

**Tech Stack:** Go 1.22, `github.com/TencentBlueKing/bk-plugin-framework-go`, `github.com/TencentBlueKing/blueapps-go`, Gin, GORM, SQLite for tests, MySQL in deployment, Cobra, Logrus adapter for framework executor.

---

## Scope

This plan implements Phase 1 from the design spec:

- legacy `hub.MustInstall(...)` remains compatible
- new `hub.MustInstallV2(...)` is added for explicit input schemas
- new module `github.com/TencentBlueKing/bk-plugin-runtime-go`
- `runner.Run()`
- commands: `server`, `worker`, `syncapigw`, `collectstatics`, `version`
- routes: `GET /bk_plugin/meta`, `GET /bk_plugin/detail/:version`, `POST /bk_plugin/invoke/:version`, `GET /bk_plugin/schedule/:trace_id`
- durable schedule store
- DB-backed worker for poll tasks
- migration guide and one legacy fixture

Separate plans cover callback, full APIGW authorization, plugin finish callback, production APIGW resource synchronization, and plugin custom API dispatch.

## File Map

Modify in current repo `bk-plugin-framework-go`:

- `hub/registry.go`: add `PluginSpec`, `MustInstallV2`, input schema storage, and form accessors.
- `hub/registry_test.go`: add tests for legacy compatibility and `MustInstallV2`.
- `docs/migration/beego-runtime-to-runtime-go.md`: migration guide.

Create sibling repo `../bk-plugin-runtime-go`:

- `go.mod`: runtime module dependencies.
- `runner/runner.go`: public runtime entrypoint.
- `cmd/root.go`: command root and default command behavior.
- `cmd/server.go`: server command.
- `cmd/worker.go`: worker command.
- `cmd/syncapigw.go`: compatibility command.
- `cmd/collectstatics.go`: compatibility command.
- `cmd/version.go`: version command.
- `internal/version/version.go`: runtime version value.
- `internal/blueappsadapter/bootstrap.go`: blueapps-go initialization through public APIs.
- `internal/httpx/response.go`: response envelope helpers.
- `internal/store/model.go`: schedule model and interfaces.
- `internal/store/json.go`: JSON encoding helpers.
- `internal/store/gorm_store.go`: GORM schedule store.
- `internal/store/gorm_store_test.go`: store tests.
- `internal/runtimeadapter/reader.go`: executor context reader.
- `internal/runtimeadapter/object_store.go`: context and output object stores.
- `internal/runtimeadapter/execute_runtime.go`: executor runtime implementation.
- `internal/server/router.go`: Gin router setup for plugin protocol.
- `internal/server/handlers.go`: meta/detail/invoke/schedule handlers.
- `internal/server/handlers_test.go`: HTTP protocol tests.
- `internal/scheduler/worker.go`: DB-backed poll worker.
- `internal/scheduler/worker_test.go`: worker tests.
- `examples/legacy-compatible-plugin/main.go`: fixture plugin.
- `docs/migration/beego-runtime-to-runtime-go.md`: runtime-side copy of migration instructions.

## Task 1: Add Explicit PluginSpec Registration To Framework

**Files:**
- Modify: `hub/registry.go`
- Modify: `hub/registry_test.go`

- [ ] **Step 1: Write failing tests for legacy behavior and `MustInstallV2`**

Add these tests to `hub/registry_test.go`:

```go
func TestMustInstallLegacyKeepsInputsFormAsInputsSchema(t *testing.T) {
	clearHub()

	inputsForm := []byte(`{"template_id":{"type":"int","required":true}}`)
	MustInstall(&MustInstallTestPlugin{version: "2.0.0"}, MustInstallTestPluginContextInput{}, MustInstallTestPluginOutput{}, inputsForm)

	detail, err := GetPluginDetail("2.0.0")
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"template_id": map[string]interface{}{
			"type":     "int",
			"required": true,
		},
	}, detail.InputsSchemaJSON())
	assert.Equal(t, detail.FormsRenderFormJSON(), detail.InputsSchemaJSON())
}

func TestMustInstallV2StoresExplicitSchemasAndForm(t *testing.T) {
	clearHub()

	type Inputs struct {
		TemplateID int    `json:"template_id"`
		TaskName   string `json:"task_name"`
	}
	type ContextInputs struct {
		BizID int `json:"bk_biz_id"`
	}
	type Outputs struct {
		Result string `json:"result"`
	}

	form := []byte(`{"template_id":{"component":"input-number"},"task_name":{"component":"input"}}`)
	MustInstallV2(&MustInstallTestPlugin{version: "2.1.0"}, PluginSpec{
		Inputs:        Inputs{},
		ContextInputs: ContextInputs{},
		Outputs:       Outputs{},
		Form:          form,
	})

	detail, err := GetPluginDetail("2.1.0")
	assert.Nil(t, err)
	assert.Contains(t, detail.InputsSchemaJSON()["properties"], "template_id")
	assert.Contains(t, detail.ContextInputsSchemaJSON()["properties"], "bk_biz_id")
	assert.Contains(t, detail.OutputsSchemaJSON()["properties"], "result")
	assert.Equal(t, map[string]interface{}{
		"template_id": map[string]interface{}{"component": "input-number"},
		"task_name":   map[string]interface{}{"component": "input"},
	}, detail.FormsRenderFormJSON())
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./hub -run 'TestMustInstallLegacyKeepsInputsFormAsInputsSchema|TestMustInstallV2StoresExplicitSchemasAndForm' -count=1
```

Expected: FAIL because `MustInstallV2` and `FormsRenderFormJSON` are not defined.

- [ ] **Step 3: Implement `PluginSpec`, `MustInstallV2`, and form accessor**

Modify `hub/registry.go` with these concrete changes:

```go
type PluginSpec struct {
	Inputs        interface{}
	ContextInputs interface{}
	Outputs       interface{}
	Form          []byte
}

type PluginDetail struct {
	plugin                  kit.Plugin
	inputsSchema            []byte
	contextInputsSchema     []byte
	outputsSchema           []byte
	inputsSchemaJSON        map[string]interface{}
	contextInputsSchemaJSON map[string]interface{}
	outputsSchemaJSON       map[string]interface{}
	formsRenderFormJSON     map[string]interface{}
}

func (p *PluginDetail) InputsSchema() []byte {
	return p.inputsSchema
}

func (p *PluginDetail) FormsRenderFormJSON() map[string]interface{} {
	return p.formsRenderFormJSON
}

func mustInstallDetail(p kit.Plugin, spec PluginSpec, legacyInputsFormAsSchema bool) {
	v := p.Version()
	if !versionRe.MatchString(v) {
		panic(fmt.Errorf("%s is not a valid plugin version\n", v))
	}
	if _, found := hub[v]; found {
		panic(fmt.Errorf("version %v already been installed\n", v))
	}

	inputsSchema, inputsSchemaJSON, err := reflectJSONSchema(spec.Inputs, nil)
	if err != nil {
		panic(err)
	}
	contextInputsSchema, contextInputsSchemaJSON, err := reflectJSONSchema(spec.ContextInputs, nil)
	if err != nil {
		panic(err)
	}
	outputsSchema, outputsSchemaJSON, err := reflectJSONSchema(spec.Outputs, nil)
	if err != nil {
		panic(err)
	}

	formsRenderFormJSON := map[string]interface{}{}
	if len(spec.Form) > 0 {
		if err := json.Unmarshal(spec.Form, &formsRenderFormJSON); err != nil {
			panic(err)
		}
	}
	if legacyInputsFormAsSchema {
		inputsSchema = spec.Form
		inputsSchemaJSON = formsRenderFormJSON
	}

	hub[v] = &PluginDetail{
		plugin:                  p,
		inputsSchema:            inputsSchema,
		contextInputsSchema:     contextInputsSchema,
		outputsSchema:           outputsSchema,
		inputsSchemaJSON:        inputsSchemaJSON,
		contextInputsSchemaJSON: contextInputsSchemaJSON,
		outputsSchemaJSON:       outputsSchemaJSON,
		formsRenderFormJSON:     formsRenderFormJSON,
	}
}

func MustInstall(p kit.Plugin, contextInputs interface{}, outputs interface{}, InputsForm []byte) {
	mustInstallDetail(p, PluginSpec{
		ContextInputs: contextInputs,
		Outputs:       outputs,
		Form:          InputsForm,
	}, true)
}

func MustInstallV2(p kit.Plugin, spec PluginSpec) {
	mustInstallDetail(p, spec, false)
}
```

Update the existing `TestPluginDetailPlugin` fixture construction to include `inputsSchema` and `formsRenderFormJSON`:

```go
inputsSchema := []byte("{\"inputsSchema\": 1}")
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
}

assert.Equal(t, detail.InputsSchema(), inputsSchema)
assert.Equal(t, detail.FormsRenderFormJSON(), formsRenderFormJSON)
```

- [ ] **Step 4: Run hub tests and verify pass**

Run:

```bash
go test ./hub -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full framework tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS. Empty executor tests stay harmless until they are filled by a separate framework test cleanup.

- [ ] **Step 6: Commit framework SDK change**

Run:

```bash
git add hub/registry.go hub/registry_test.go
git commit -m "feat: add explicit plugin registration spec"
```

## Task 2: Create Runtime Module Scaffold

**Files:**
- Create: `../bk-plugin-runtime-go/go.mod`
- Create: `../bk-plugin-runtime-go/runner/runner.go`
- Create: `../bk-plugin-runtime-go/cmd/root.go`
- Create: `../bk-plugin-runtime-go/cmd/server.go`
- Create: `../bk-plugin-runtime-go/cmd/worker.go`
- Create: `../bk-plugin-runtime-go/cmd/syncapigw.go`
- Create: `../bk-plugin-runtime-go/cmd/collectstatics.go`
- Create: `../bk-plugin-runtime-go/cmd/version.go`
- Create: `../bk-plugin-runtime-go/internal/version/version.go`

- [ ] **Step 1: Create sibling runtime repo directory**

Run:

```bash
mkdir -p ../bk-plugin-runtime-go
cd ../bk-plugin-runtime-go
git init
```

Expected: a new Git repository exists at `/Users/dengyh/Projects/bk-plugin-runtime-go`.

- [ ] **Step 2: Create runtime `go.mod`**

Create `../bk-plugin-runtime-go/go.mod`:

```go
module github.com/TencentBlueKing/bk-plugin-runtime-go

go 1.22

require (
	github.com/TencentBlueKing/bk-plugin-framework-go v0.5.0
	github.com/TencentBlueKing/blueapps-go v1.6.2
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.10.0
	gorm.io/datatypes v1.2.1
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.25.12
)

replace github.com/TencentBlueKing/bk-plugin-framework-go => ../bk-plugin-framework-go
```

- [ ] **Step 3: Create public runner**

Create `../bk-plugin-runtime-go/runner/runner.go`:

```go
package runner

import "github.com/TencentBlueKing/bk-plugin-runtime-go/cmd"

func Run() {
	cmd.Execute()
}
```

- [ ] **Step 4: Create command root**

Create `../bk-plugin-runtime-go/cmd/root.go`:

```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bk-plugin-runtime-go",
	Short: "Run a BlueKing plugin runtime process",
}

func Execute() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "server")
	}
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
```

- [ ] **Step 5: Create version package and command**

Create `../bk-plugin-runtime-go/internal/version/version.go`:

```go
package version

const Version = "0.1.0"
```

Create `../bk-plugin-runtime-go/cmd/version.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/version"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print runtime version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version)
		},
	})
}
```

- [ ] **Step 6: Create compatibility commands**

Create `../bk-plugin-runtime-go/cmd/server.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "server",
		Short: "Start plugin HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("server command registered")
		},
	})
}
```

Create `../bk-plugin-runtime-go/cmd/worker.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "worker",
		Short: "Start plugin schedule worker",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("worker command registered")
		},
	})
}
```

Create `../bk-plugin-runtime-go/cmd/syncapigw.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "syncapigw",
		Short: "Synchronize APIGW resources",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("syncapigw command is not active in phase 1")
		},
	})
}
```

Create `../bk-plugin-runtime-go/cmd/collectstatics.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "collectstatics",
		Short: "Compatibility no-op for old beego-runtime deployments",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("collectstatics is a no-op in bk-plugin-runtime-go")
		},
	})
}
```

- [ ] **Step 7: Verify runtime module compiles**

Run:

```bash
cd ../bk-plugin-runtime-go
go mod tidy
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit runtime scaffold**

Run:

```bash
cd ../bk-plugin-runtime-go
git add .
git commit -m "feat: add plugin runtime command scaffold"
```

## Task 3: Add Blueapps Bootstrap And HTTP Response Helpers

**Files:**
- Create: `../bk-plugin-runtime-go/internal/blueappsadapter/bootstrap.go`
- Create: `../bk-plugin-runtime-go/internal/httpx/response.go`
- Test through: `go test ./...`

- [ ] **Step 1: Create response helper**

Create `../bk-plugin-runtime-go/internal/httpx/response.go`:

```go
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "OK", Data: data})
}

func Error(c *gin.Context, status int, code int, message string) {
	c.JSON(status, Envelope{Code: code, Message: message, Data: nil})
}
```

- [ ] **Step 2: Create blueapps bootstrap adapter**

Create `../bk-plugin-runtime-go/internal/blueappsadapter/bootstrap.go`:

```go
package blueappsadapter

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueapps-go/pkg/cache/memory"
	"github.com/TencentBlueKing/blueapps-go/pkg/config"
	"github.com/TencentBlueKing/blueapps-go/pkg/i18n"
	"github.com/TencentBlueKing/blueapps-go/pkg/infras/database"
	"github.com/TencentBlueKing/blueapps-go/pkg/infras/redis"
	log "github.com/TencentBlueKing/blueapps-go/pkg/logging"
)

func LoadAndInit(ctx context.Context, cfgFile string) (*config.Config, error) {
	cfg, err := config.Load(ctx, cfgFile)
	if err != nil {
		return nil, errors.Wrap(err, "load blueapps config")
	}
	i18n.InitMsgMap()
	if err := initLoggers(&cfg.Service.Log); err != nil {
		return nil, err
	}
	database.InitDBClient(ctx, cfg.Platform.Addons.Mysql, log.GetLogger("gorm"))
	if cfg.Platform.Addons.Redis != nil {
		redis.InitRedisClient(ctx, cfg.Platform.Addons.Redis)
	}
	memory.InitCache(cfg.Service.MemoryCacheSize)
	return cfg, nil
}

func initLoggers(cfg *config.LogConfig) error {
	if err := os.MkdirAll(cfg.Dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "create log dir %s", cfg.Dir)
	}
	writerName := "file"
	if cfg.ForceToStdout {
		writerName = "stdout"
	}
	if err := initLogger("default", cfg.Level, lo.Ternary(writerName == "stdout", "text", "json"), writerName, filepath.Join(cfg.Dir, "default.log")); err != nil {
		return err
	}
	if err := initLogger("gorm", log.GormLogLevel, "json", "file", filepath.Join(cfg.Dir, "gorm.log")); err != nil {
		return err
	}
	return initLogger("gin", log.GinLogLevel, "json", "file", filepath.Join(cfg.Dir, "gin.log"))
}

func initLogger(name string, level string, handler string, writer string, filename string) error {
	return log.InitLogger(name, &log.Options{
		Level:        level,
		HandlerName:  handler,
		WriterName:   writer,
		WriterConfig: map[string]string{"filename": filename},
	})
}
```

- [ ] **Step 3: Run tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit bootstrap helpers**

Run:

```bash
cd ../bk-plugin-runtime-go
git add internal/blueappsadapter internal/httpx go.mod go.sum
git commit -m "feat: add blueapps bootstrap adapter"
```

## Task 4: Implement Durable Schedule Store

**Files:**
- Create: `../bk-plugin-runtime-go/internal/store/model.go`
- Create: `../bk-plugin-runtime-go/internal/store/json.go`
- Create: `../bk-plugin-runtime-go/internal/store/gorm_store.go`
- Create: `../bk-plugin-runtime-go/internal/store/gorm_store_test.go`

- [ ] **Step 1: Write failing store tests**

Create `../bk-plugin-runtime-go/internal/store/gorm_store_test.go`:

```go
package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
)

func newTestStore(t *testing.T) *GormStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Schedule{}))
	return NewGormStore(db)
}

func TestGormStoreCreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	err := s.Create(ctx, &Schedule{
		TraceID:       "trace-1",
		PluginVersion: "1.0.0",
		State:         constants.StatePoll,
		InvokeCount:   1,
		Inputs:        JSONMap{"x": float64(1)},
		ContextInputs: JSONMap{"bk_biz_id": float64(2)},
		NextRunAt:     time.Now().UTC(),
	})
	require.NoError(t, err)

	got, err := s.Get(ctx, "trace-1")
	require.NoError(t, err)
	require.Equal(t, "1.0.0", got.PluginVersion)
	require.Equal(t, constants.StatePoll, got.State)
	require.Equal(t, JSONMap{"x": float64(1)}, got.Inputs)
}

func TestGormStoreClaimDueSkipsLockedRows(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	now := time.Now().UTC()

	require.NoError(t, s.Create(ctx, &Schedule{TraceID: "due", PluginVersion: "1.0.0", State: constants.StatePoll, InvokeCount: 1, NextRunAt: now.Add(-time.Second)}))
	require.NoError(t, s.Create(ctx, &Schedule{TraceID: "future", PluginVersion: "1.0.0", State: constants.StatePoll, InvokeCount: 1, NextRunAt: now.Add(time.Hour)}))
	require.NoError(t, s.Create(ctx, &Schedule{TraceID: "done", PluginVersion: "1.0.0", State: constants.StateSuccess, InvokeCount: 1, FinishedAt: ptrTime(now)}))

	claimed, err := s.ClaimDue(ctx, now, "worker-a", 5, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, "due", claimed[0].TraceID)

	claimedAgain, err := s.ClaimDue(ctx, now, "worker-b", 5, time.Minute)
	require.NoError(t, err)
	require.Empty(t, claimedAgain)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/store -count=1
```

Expected: FAIL because store types do not exist.

- [ ] **Step 3: Implement JSON map type**

Create `../bk-plugin-runtime-go/internal/store/json.go`:

```go
package store

import (
	"database/sql/driver"
	"encoding/json"
)

type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data = []byte(`{}`)
	}
	return json.Unmarshal(data, m)
}

func ToJSONMap(raw []byte) (JSONMap, error) {
	if len(raw) == 0 {
		return JSONMap{}, nil
	}
	var data JSONMap
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}
```

- [ ] **Step 4: Implement schedule model and interface**

Create `../bk-plugin-runtime-go/internal/store/model.go`:

```go
package store

import (
	"context"
	"time"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
)

type Schedule struct {
	ID            uint            `gorm:"primaryKey"`
	TraceID       string          `gorm:"size:64;uniqueIndex;not null"`
	PluginVersion string          `gorm:"size:32;index;not null"`
	State         constants.State `gorm:"index;not null"`
	InvokeCount   int             `gorm:"not null"`
	Inputs        JSONMap         `gorm:"type:json"`
	ContextInputs JSONMap         `gorm:"type:json"`
	ContextData   JSONMap         `gorm:"type:json"`
	Outputs       JSONMap         `gorm:"type:json"`
	ErrorCode     string          `gorm:"size:64"`
	ErrorMessage  string          `gorm:"type:text"`
	ErrorDetail   string          `gorm:"type:text"`
	NextRunAt     time.Time       `gorm:"index"`
	LockedBy      string          `gorm:"size:128;index"`
	LockedUntil   *time.Time      `gorm:"index"`
	FinishedAt    *time.Time      `gorm:"index"`
	CallerApp     string          `gorm:"size:64"`
	Operator      string          `gorm:"size:64"`
	RequestID      string          `gorm:"size:128"`
	TenantID       string          `gorm:"size:64"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ScheduleStore interface {
	Create(ctx context.Context, schedule *Schedule) error
	Get(ctx context.Context, traceID string) (*Schedule, error)
	UpdateContextData(ctx context.Context, traceID string, data JSONMap) error
	UpdateOutputs(ctx context.Context, traceID string, data JSONMap) error
	MarkPoll(ctx context.Context, traceID string, invokeCount int, nextRunAt time.Time) error
	MarkSuccess(ctx context.Context, traceID string) error
	MarkFail(ctx context.Context, traceID string, message string) error
	ClaimDue(ctx context.Context, now time.Time, workerID string, limit int, lockFor time.Duration) ([]Schedule, error)
}
```

- [ ] **Step 5: Implement GORM store**

Create `../bk-plugin-runtime-go/internal/store/gorm_store.go`:

```go
package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) AutoMigrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&Schedule{})
}

func (s *GormStore) Create(ctx context.Context, schedule *Schedule) error {
	if schedule.ContextData == nil {
		schedule.ContextData = JSONMap{}
	}
	if schedule.Outputs == nil {
		schedule.Outputs = JSONMap{}
	}
	return s.db.WithContext(ctx).Create(schedule).Error
}

func (s *GormStore) Get(ctx context.Context, traceID string) (*Schedule, error) {
	var schedule Schedule
	if err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).First(&schedule).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *GormStore) UpdateContextData(ctx context.Context, traceID string, data JSONMap) error {
	return s.db.WithContext(ctx).Model(&Schedule{}).Where("trace_id = ?", traceID).Update("context_data", data).Error
}

func (s *GormStore) UpdateOutputs(ctx context.Context, traceID string, data JSONMap) error {
	return s.db.WithContext(ctx).Model(&Schedule{}).Where("trace_id = ?", traceID).Update("outputs", data).Error
}

func (s *GormStore) MarkPoll(ctx context.Context, traceID string, invokeCount int, nextRunAt time.Time) error {
	return s.db.WithContext(ctx).Model(&Schedule{}).Where("trace_id = ?", traceID).Updates(map[string]interface{}{
		"state":        constants.StatePoll,
		"invoke_count": invokeCount,
		"next_run_at":  nextRunAt,
		"locked_by":    "",
		"locked_until": nil,
	}).Error
}

func (s *GormStore) MarkSuccess(ctx context.Context, traceID string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Schedule{}).Where("trace_id = ?", traceID).Updates(map[string]interface{}{
		"state":        constants.StateSuccess,
		"finished_at":  &now,
		"locked_by":    "",
		"locked_until": nil,
	}).Error
}

func (s *GormStore) MarkFail(ctx context.Context, traceID string, message string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Schedule{}).Where("trace_id = ?", traceID).Updates(map[string]interface{}{
		"state":         constants.StateFail,
		"error_code":    "PLUGIN_EXECUTE_ERROR",
		"error_message": message,
		"finished_at":   &now,
		"locked_by":     "",
		"locked_until":  nil,
	}).Error
}

func (s *GormStore) ClaimDue(ctx context.Context, now time.Time, workerID string, limit int, lockFor time.Duration) ([]Schedule, error) {
	var candidates []Schedule
	err := s.db.WithContext(ctx).
		Where("state = ?", constants.StatePoll).
		Where("finished_at IS NULL").
		Where("next_run_at <= ?", now).
		Where("locked_until IS NULL OR locked_until < ?", now).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	claimed := make([]Schedule, 0, len(candidates))
	lockUntil := now.Add(lockFor)
	for _, item := range candidates {
		result := s.db.WithContext(ctx).Model(&Schedule{}).
			Where("trace_id = ?", item.TraceID).
			Where("locked_until IS NULL OR locked_until < ?", now).
			Updates(map[string]interface{}{"locked_by": workerID, "locked_until": &lockUntil})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			item.LockedBy = workerID
			item.LockedUntil = &lockUntil
			claimed = append(claimed, item)
		}
	}
	return claimed, nil
}
```

- [ ] **Step 6: Run store tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/store -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit store**

Run:

```bash
cd ../bk-plugin-runtime-go
git add internal/store go.mod go.sum
git commit -m "feat: add durable plugin schedule store"
```

## Task 5: Add Executor Runtime Adapter

**Files:**
- Create: `../bk-plugin-runtime-go/internal/runtimeadapter/reader.go`
- Create: `../bk-plugin-runtime-go/internal/runtimeadapter/object_store.go`
- Create: `../bk-plugin-runtime-go/internal/runtimeadapter/execute_runtime.go`
- Test through: `internal/server/handlers_test.go` in Task 6

- [ ] **Step 1: Implement context reader**

Create `../bk-plugin-runtime-go/internal/runtimeadapter/reader.go`:

```go
package runtimeadapter

import (
	"encoding/json"

	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type Reader struct {
	Inputs        store.JSONMap
	ContextInputs store.JSONMap
}

func (r Reader) ReadInputs(v interface{}) error {
	data, err := json.Marshal(r.Inputs)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (r Reader) ReadContextInputs(v interface{}) error {
	data, err := json.Marshal(r.ContextInputs)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
```

- [ ] **Step 2: Implement object store adapter**

Create `../bk-plugin-runtime-go/internal/runtimeadapter/object_store.go`:

```go
package runtimeadapter

import (
	"context"
	"encoding/json"

	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type Field string

const (
	FieldContextData Field = "context_data"
	FieldOutputs     Field = "outputs"
)

type ObjectStore struct {
	ctx   context.Context
	store store.ScheduleStore
	field Field
}

func NewObjectStore(ctx context.Context, scheduleStore store.ScheduleStore, field Field) *ObjectStore {
	return &ObjectStore{ctx: ctx, store: scheduleStore, field: field}
}

func (s *ObjectStore) Write(traceID string, v interface{}) error {
	data, err := toJSONMap(v)
	if err != nil {
		return err
	}
	switch s.field {
	case FieldContextData:
		return s.store.UpdateContextData(s.ctx, traceID, data)
	case FieldOutputs:
		return s.store.UpdateOutputs(s.ctx, traceID, data)
	default:
		return nil
	}
}

func (s *ObjectStore) Read(traceID string, v interface{}) error {
	schedule, err := s.store.Get(s.ctx, traceID)
	if err != nil {
		return err
	}
	var data store.JSONMap
	switch s.field {
	case FieldContextData:
		data = schedule.ContextData
	case FieldOutputs:
		data = schedule.Outputs
	default:
		data = store.JSONMap{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func toJSONMap(v interface{}) (store.JSONMap, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return store.ToJSONMap(raw)
}
```

- [ ] **Step 3: Implement execute runtime adapter**

Create `../bk-plugin-runtime-go/internal/runtimeadapter/execute_runtime.go`:

```go
package runtimeadapter

import (
	"context"
	"time"

	"github.com/TencentBlueKing/bk-plugin-framework-go/runtime"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type ExecuteRuntime struct {
	ctx   context.Context
	store store.ScheduleStore
}

func NewExecuteRuntime(ctx context.Context, scheduleStore store.ScheduleStore) *ExecuteRuntime {
	return &ExecuteRuntime{ctx: ctx, store: scheduleStore}
}

func (r *ExecuteRuntime) GetOutputsStore() runtime.ObjectStore {
	return NewObjectStore(r.ctx, r.store, FieldOutputs)
}

func (r *ExecuteRuntime) GetContextStore() runtime.ObjectStore {
	return NewObjectStore(r.ctx, r.store, FieldContextData)
}

func (r *ExecuteRuntime) SetPoll(traceID string, version string, invokeCount int, after time.Duration) error {
	return r.store.MarkPoll(r.ctx, traceID, invokeCount, time.Now().UTC().Add(after))
}

func (r *ExecuteRuntime) SetFail(traceID string, err error) error {
	return r.store.MarkFail(r.ctx, traceID, err.Error())
}

func (r *ExecuteRuntime) SetSuccess(traceID string) error {
	return r.store.MarkSuccess(r.ctx, traceID)
}
```

- [ ] **Step 4: Run compile check**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/runtimeadapter -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit adapter**

Run:

```bash
cd ../bk-plugin-runtime-go
git add internal/runtimeadapter
git commit -m "feat: add framework executor runtime adapter"
```

## Task 6: Implement Meta Detail Invoke Schedule HTTP Protocol

**Files:**
- Create: `../bk-plugin-runtime-go/internal/server/router.go`
- Create: `../bk-plugin-runtime-go/internal/server/handlers.go`
- Create: `../bk-plugin-runtime-go/internal/server/handlers_test.go`
- Modify: `../bk-plugin-runtime-go/cmd/server.go`

- [ ] **Step 1: Write failing HTTP handler tests**

Create `../bk-plugin-runtime-go/internal/server/handlers_test.go`:

```go
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type testPlugin struct{}

func (p testPlugin) Version() string { return "9.9.1" }
func (p testPlugin) Desc() string    { return "test plugin" }
func (p testPlugin) Execute(ctx *kit.Context) error {
	var inputs struct {
		Mode string `json:"mode"`
	}
	if err := ctx.ReadInputs(&inputs); err != nil {
		return err
	}
	if inputs.Mode == "poll" && ctx.InvokeCount() == 1 {
		ctx.WaitPoll(time.Millisecond)
		return nil
	}
	return ctx.WriteOutputs(map[string]interface{}{"mode": inputs.Mode, "count": ctx.InvokeCount()})
}

var installTestPluginOnce sync.Once

func newTestRouter(t *testing.T) (*gin.Engine, *store.GormStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	s := store.NewGormStore(db)
	require.NoError(t, s.AutoMigrate(context.Background()))
	installTestPluginOnce.Do(func() {
		hub.MustInstallV2(testPlugin{}, hub.PluginSpec{
			Inputs:  struct{ Mode string `json:"mode"` }{},
			Outputs: struct{ Mode string `json:"mode"` }{},
			Form:    []byte(`{"mode":{"component":"input"}}`),
		})
	})
	return NewRouter(Config{Store: s, Logger: logrus.NewEntry(logrus.StandardLogger())}), s
}

func TestMetaAndDetail(t *testing.T) {
	router, _ := newTestRouter(t)

	meta := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bk_plugin/meta", nil)
	router.ServeHTTP(meta, req)
	require.Equal(t, http.StatusOK, meta.Code)
	require.Contains(t, meta.Body.String(), "9.9.1")

	detail := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/bk_plugin/detail/9.9.1", nil)
	router.ServeHTTP(detail, req)
	require.Equal(t, http.StatusOK, detail.Code)
	require.Contains(t, detail.Body.String(), "renderform")
	require.Contains(t, detail.Body.String(), "test plugin")
}

func TestInvokeSyncAndScheduleRead(t *testing.T) {
	router, _ := newTestRouter(t)

	body := bytes.NewBufferString(`{"inputs":{"mode":"sync"},"context":{}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bk_plugin/invoke/9.9.1", body)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload struct {
		Data struct {
			TraceID string `json:"trace_id"`
			State   int    `json:"state"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.Data.TraceID)
	require.Equal(t, 4, payload.Data.State)

	schedule := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/bk_plugin/schedule/"+payload.Data.TraceID, nil)
	router.ServeHTTP(schedule, req)
	require.Equal(t, http.StatusOK, schedule.Code)
	require.Contains(t, schedule.Body.String(), `"mode":"sync"`)
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/server -count=1
```

Expected: FAIL because `NewRouter`, `Config`, and handlers do not exist.

- [ ] **Step 3: Implement router**

Create `../bk-plugin-runtime-go/internal/server/router.go`:

```go
package server

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type Config struct {
	Store  store.ScheduleStore
	Logger *logrus.Entry
}

func NewRouter(cfg Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if cfg.Logger == nil {
		cfg.Logger = logrus.NewEntry(logrus.StandardLogger())
	}

	h := Handler{store: cfg.Store, logger: cfg.Logger}
	group := r.Group("/bk_plugin")
	group.GET("/meta", h.Meta)
	group.GET("/detail/:version", h.Detail)
	group.POST("/invoke/:version", h.Invoke)
	group.GET("/schedule/:trace_id", h.Schedule)
	return r
}
```

- [ ] **Step 4: Implement handlers**

Create `../bk-plugin-runtime-go/internal/server/handlers.go`:

```go
package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/executor"
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/httpx"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/runtimeadapter"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/version"
)

type Handler struct {
	store  store.ScheduleStore
	logger *logrus.Entry
}

type invokeRequest struct {
	Inputs  store.JSONMap `json:"inputs"`
	Context store.JSONMap `json:"context"`
}

func (h Handler) Meta(c *gin.Context) {
	httpx.OK(c, gin.H{
		"language":        "go",
		"runtime_version": version.Version,
		"versions":        hub.GetPluginVersions(),
	})
}

func (h Handler) Detail(c *gin.Context) {
	detail, err := hub.GetPluginDetail(c.Param("version"))
	if err != nil {
		httpx.Error(c, http.StatusNotFound, 40404, err.Error())
		return
	}
	httpx.OK(c, gin.H{
		"version":        detail.Plugin().Version(),
		"desc":           detail.Plugin().Desc(),
		"inputs":         detail.InputsSchemaJSON(),
		"context_inputs": detail.ContextInputsSchemaJSON(),
		"outputs":        detail.OutputsSchemaJSON(),
		"forms": gin.H{
			"renderform": detail.FormsRenderFormJSON(),
		},
	})
}

func (h Handler) Invoke(c *gin.Context) {
	var req invokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	traceID := uuid.NewString()
	versionCode := c.Param("version")
	schedule := &store.Schedule{
		TraceID:       traceID,
		PluginVersion: versionCode,
		State:         constants.StateEmpty,
		InvokeCount:   1,
		Inputs:        req.Inputs,
		ContextInputs: req.Context,
		ContextData:   store.JSONMap{},
		Outputs:       store.JSONMap{},
		NextRunAt:     time.Now().UTC(),
	}
	if err := h.store.Create(c.Request.Context(), schedule); err != nil {
		httpx.Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	reader := runtimeadapter.Reader{Inputs: req.Inputs, ContextInputs: req.Context}
	rt := runtimeadapter.NewExecuteRuntime(c.Request.Context(), h.store)
	logger := h.logger.WithField("trace_id", traceID)
	state, err := executor.Execute(traceID, versionCode, reader, rt, logger)
	if err != nil {
		_ = h.store.MarkFail(c.Request.Context(), traceID, err.Error())
		httpx.OK(c, gin.H{"trace_id": traceID, "state": constants.StateFail})
		return
	}
	if state == constants.StateSuccess {
		_ = h.store.MarkSuccess(c.Request.Context(), traceID)
	}
	saved, _ := h.store.Get(c.Request.Context(), traceID)
	httpx.OK(c, gin.H{"trace_id": traceID, "state": saved.State, "outputs": saved.Outputs})
}

func (h Handler) Schedule(c *gin.Context) {
	schedule, err := h.store.Get(c.Request.Context(), c.Param("trace_id"))
	if err != nil {
		httpx.Error(c, http.StatusNotFound, 40404, err.Error())
		return
	}
	httpx.OK(c, gin.H{
		"trace_id": schedule.TraceID,
		"version":  schedule.PluginVersion,
		"state":    schedule.State,
		"outputs":  schedule.Outputs,
		"error": gin.H{
			"code":    schedule.ErrorCode,
			"message": schedule.ErrorMessage,
		},
	})
}
```

- [ ] **Step 5: Update server command to start router**

Replace `../bk-plugin-runtime-go/cmd/server.go` with:

```go
package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueapps-go/pkg/config"
	"github.com/TencentBlueKing/blueapps-go/pkg/infras/database"
	log "github.com/TencentBlueKing/blueapps-go/pkg/logging"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/blueappsadapter"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/server"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

func init() {
	var cfgFile string
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start plugin HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			cfg, err := blueappsadapter.LoadAndInit(ctx, cfgFile)
			if err != nil {
				log.Fatalf("init runtime: %s", err)
			}
			scheduleStore := store.NewGormStore(database.Client(ctx))
			if err := scheduleStore.AutoMigrate(ctx); err != nil {
				log.Fatalf("migrate plugin schedules: %s", err)
			}
			srv := &http.Server{
				Addr:    ":" + strconv.Itoa(config.G.Service.Server.Port),
				Handler: server.NewRouter(server.Config{Store: scheduleStore}),
			}
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("start server: %s", err)
				}
			}()
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit
			shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Service.Server.GraceTimeout)*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Fatalf("shutdown server: %s", err)
			}
		},
	}
	cmd.Flags().StringVar(&cfgFile, "conf", "", "config file")
	rootCmd.AddCommand(cmd)
}
```

- [ ] **Step 6: Run server tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/server -count=1
```

Expected: PASS.

- [ ] **Step 7: Run all runtime tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit HTTP protocol**

Run:

```bash
cd ../bk-plugin-runtime-go
git add cmd/server.go internal/server go.mod go.sum
git commit -m "feat: add plugin HTTP protocol"
```

## Task 7: Implement Poll Worker

**Files:**
- Create: `../bk-plugin-runtime-go/internal/scheduler/worker.go`
- Create: `../bk-plugin-runtime-go/internal/scheduler/worker_test.go`
- Modify: `../bk-plugin-runtime-go/cmd/worker.go`

- [ ] **Step 1: Write failing worker test**

Create `../bk-plugin-runtime-go/internal/scheduler/worker_test.go`:

```go
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type pollPlugin struct{}

func (p pollPlugin) Version() string { return "9.9.2" }
func (p pollPlugin) Desc() string    { return "poll plugin" }
func (p pollPlugin) Execute(ctx *kit.Context) error {
	if ctx.InvokeCount() == 1 {
		ctx.WaitPoll(time.Millisecond)
		return nil
	}
	return ctx.WriteOutputs(map[string]interface{}{"done": true, "count": ctx.InvokeCount()})
}

func TestWorkerRunsDuePollTask(t *testing.T) {
	ctx := context.Background()
	hub.MustInstallV2(pollPlugin{}, hub.PluginSpec{Form: []byte(`{}`)})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	s := store.NewGormStore(db)
	require.NoError(t, s.AutoMigrate(ctx))
	require.NoError(t, s.Create(ctx, &store.Schedule{
		TraceID:       "poll-trace",
		PluginVersion: "9.9.2",
		State:         constants.StatePoll,
		InvokeCount:   2,
		Inputs:        store.JSONMap{},
		ContextInputs: store.JSONMap{},
		ContextData:   store.JSONMap{},
		Outputs:       store.JSONMap{},
		NextRunAt:     time.Now().UTC().Add(-time.Second),
	}))

	worker := NewWorker(Config{
		Store:    s,
		WorkerID: "test-worker",
		Limit:    10,
		LockFor:  time.Minute,
		Logger:   logrus.NewEntry(logrus.StandardLogger()),
	})
	require.NoError(t, worker.RunOnce(ctx))

	got, err := s.Get(ctx, "poll-trace")
	require.NoError(t, err)
	require.Equal(t, constants.StateSuccess, got.State)
	require.Equal(t, store.JSONMap{"done": true, "count": float64(2)}, got.Outputs)
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/scheduler -count=1
```

Expected: FAIL because worker does not exist.

- [ ] **Step 3: Implement worker**

Create `../bk-plugin-runtime-go/internal/scheduler/worker.go`:

```go
package scheduler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/TencentBlueKing/bk-plugin-framework-go/executor"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/runtimeadapter"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

type Config struct {
	Store    store.ScheduleStore
	WorkerID string
	Limit    int
	LockFor  time.Duration
	Interval time.Duration
	Logger   *logrus.Entry
}

type Worker struct {
	cfg Config
}

func NewWorker(cfg Config) *Worker {
	if cfg.Limit == 0 {
		cfg.Limit = 10
	}
	if cfg.LockFor == 0 {
		cfg.LockFor = 5 * time.Minute
	}
	if cfg.Interval == 0 {
		cfg.Interval = time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &Worker{cfg: cfg}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			w.cfg.Logger.WithError(err).Error("run schedule once")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	now := time.Now().UTC()
	items, err := w.cfg.Store.ClaimDue(ctx, now, w.cfg.WorkerID, w.cfg.Limit, w.cfg.LockFor)
	if err != nil {
		return err
	}
	for _, item := range items {
		reader := runtimeadapter.Reader{Inputs: item.Inputs, ContextInputs: item.ContextInputs}
		rt := runtimeadapter.NewExecuteRuntime(ctx, w.cfg.Store)
		logger := w.cfg.Logger.WithField("trace_id", item.TraceID).WithField("plugin_version", item.PluginVersion)
		if err := executor.Schedule(item.TraceID, item.PluginVersion, item.InvokeCount, reader, rt, logger); err != nil {
			logger.WithError(err).Error("schedule plugin")
		}
	}
	return nil
}
```

- [ ] **Step 4: Update worker command**

Replace `../bk-plugin-runtime-go/cmd/worker.go` with:

```go
package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueapps-go/pkg/infras/database"
	log "github.com/TencentBlueKing/blueapps-go/pkg/logging"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/blueappsadapter"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/scheduler"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/internal/store"
)

func init() {
	var cfgFile string
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Start plugin schedule worker",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			if _, err := blueappsadapter.LoadAndInit(ctx, cfgFile); err != nil {
				log.Fatalf("init runtime: %s", err)
			}
			scheduleStore := store.NewGormStore(database.Client(ctx))
			if err := scheduleStore.AutoMigrate(ctx); err != nil {
				log.Fatalf("migrate plugin schedules: %s", err)
			}
			worker := scheduler.NewWorker(scheduler.Config{
				Store:    scheduleStore,
				WorkerID: uuid.NewString(),
				Interval: time.Second,
				Logger:   logrus.NewEntry(logrus.StandardLogger()),
			})
			if err := worker.Run(ctx); err != nil && err != context.Canceled {
				log.Fatalf("worker stopped: %s", err)
			}
		},
	}
	cmd.Flags().StringVar(&cfgFile, "conf", "", "config file")
	rootCmd.AddCommand(cmd)
}
```

- [ ] **Step 5: Run worker tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./internal/scheduler -count=1
```

Expected: PASS.

- [ ] **Step 6: Run full runtime tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit worker**

Run:

```bash
cd ../bk-plugin-runtime-go
git add cmd/worker.go internal/scheduler
git commit -m "feat: add poll schedule worker"
```

## Task 8: Add Migration Docs And Legacy Fixture

**Files:**
- Create: `docs/migration/beego-runtime-to-runtime-go.md`
- Create: `../bk-plugin-runtime-go/docs/migration/beego-runtime-to-runtime-go.md`
- Create: `../bk-plugin-runtime-go/examples/legacy-compatible-plugin/main.go`

- [ ] **Step 1: Create framework migration guide**

Create `docs/migration/beego-runtime-to-runtime-go.md` in `bk-plugin-framework-go`:

```markdown
# Migrate From beego-runtime To bk-plugin-runtime-go

## Target

This guide covers the Phase 1 migration path for existing Go plugins that use `bk-plugin-framework-go` and `beego-runtime`.

## Minimal code change

Change the runtime import in `main.go`:

```diff
- "github.com/TencentBlueKing/beego-runtime/runner"
+ "github.com/TencentBlueKing/bk-plugin-runtime-go/runner"
```

Keep plugin business code unchanged:

```go
func (p MyPlugin) Execute(ctx *kit.Context) error {
    return nil
}
```

Keep legacy registration unchanged:

```go
hub.MustInstall(MyPlugin{}, ContextInputs{}, Outputs{}, inputsForm)
```

## Dependency change

Add the runtime module:

```bash
go get github.com/TencentBlueKing/bk-plugin-runtime-go@v0.1.0
go mod tidy
```

## Process commands

Existing process commands remain valid:

```yaml
processes:
  web:
    command: ./plugin server
  worker:
    command: ./plugin worker
```

## Supported in Phase 1

- sync plugins
- poll plugins using `ctx.WaitPoll`
- `meta`, `detail`, `invoke`, and `schedule`
- durable schedule storage through the runtime database

## Requires manual migration

- direct Beego imports
- direct imports of `beego-runtime` internal packages
- Beego controller based custom plugin APIs
- behavior tied to the old debug panel UI
```

- [ ] **Step 2: Copy migration guide to runtime repo**

Run:

```bash
mkdir -p ../bk-plugin-runtime-go/docs/migration
cp docs/migration/beego-runtime-to-runtime-go.md ../bk-plugin-runtime-go/docs/migration/beego-runtime-to-runtime-go.md
```

Expected: both repos contain the same migration guide.

- [ ] **Step 3: Create legacy-compatible fixture**

Create `../bk-plugin-runtime-go/examples/legacy-compatible-plugin/main.go`:

```go
package main

import (
	_ "embed"
	"time"

	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
	"github.com/TencentBlueKing/bk-plugin-runtime-go/runner"
)

//go:embed inputs_form.json
var inputsForm []byte

type DemoPlugin struct{}

type ContextInputs struct {
	BizID int `json:"bk_biz_id"`
}

type Outputs struct {
	Message string `json:"message"`
}

func (p DemoPlugin) Version() string { return "1.0.0" }
func (p DemoPlugin) Desc() string    { return "legacy compatible plugin" }
func (p DemoPlugin) Execute(ctx *kit.Context) error {
	if ctx.InvokeCount() == 1 {
		ctx.WaitPoll(time.Second)
		return nil
	}
	return ctx.WriteOutputs(Outputs{Message: "done"})
}

func init() {
	hub.MustInstall(DemoPlugin{}, ContextInputs{}, Outputs{}, inputsForm)
}

func main() {
	runner.Run()
}
```

Create `../bk-plugin-runtime-go/examples/legacy-compatible-plugin/inputs_form.json`:

```json
{
  "template_id": {
    "component": "input-number",
    "label": "Template ID"
  }
}
```

- [ ] **Step 4: Run docs and fixture compile check**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
go test ./examples/legacy-compatible-plugin -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit framework migration guide**

Run:

```bash
cd ../bk-plugin-framework-go
git add docs/migration/beego-runtime-to-runtime-go.md
git commit -m "docs: add beego runtime migration guide"
```

- [ ] **Step 6: Commit runtime docs and fixture**

Run:

```bash
cd ../bk-plugin-runtime-go
git add docs examples go.mod go.sum
git commit -m "docs: add beego runtime migration fixture"
```

## Task 9: End-To-End Compatibility Smoke Test

**Files:**
- Use: `../bk-plugin-runtime-go/examples/legacy-compatible-plugin/main.go`
- Use: `../bk-plugin-runtime-go/internal/server/handlers_test.go`

- [ ] **Step 1: Build the fixture binary**

Run:

```bash
cd ../bk-plugin-runtime-go/examples/legacy-compatible-plugin
go build -o /tmp/legacy-compatible-plugin .
```

Expected: command exits 0 and `/tmp/legacy-compatible-plugin` exists.

- [ ] **Step 2: Run all framework tests**

Run:

```bash
cd ../bk-plugin-framework-go
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run all runtime tests**

Run:

```bash
cd ../bk-plugin-runtime-go
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Inspect final Git state**

Run:

```bash
cd ../bk-plugin-framework-go
git status --short
cd ../bk-plugin-runtime-go
git status --short
```

Expected: both commands print no changed files.
