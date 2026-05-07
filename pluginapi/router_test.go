package pluginapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingRouter struct {
	routes []recordedRoute
}

type recordedRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

func (r *recordingRouter) Handle(method string, path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, recordedRoute{method: method, path: path, handler: handler})
}

func (r *recordingRouter) GET(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodGet, path, handler)
}

func (r *recordingRouter) POST(path string, handler http.HandlerFunc) {
	r.Handle(http.MethodPost, path, handler)
}

func TestRegisterStoresHTTPRegistrars(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(func(router Router) {
		router.GET("/echo", func(w http.ResponseWriter, r *http.Request) {})
	})

	router := &recordingRouter{}
	for _, registrar := range Registrars() {
		registrar(router)
	}

	assert.Len(t, router.routes, 1)
	assert.Equal(t, http.MethodGet, router.routes[0].method)
	assert.Equal(t, "/echo", router.routes[0].path)
	assert.NotNil(t, router.routes[0].handler)
}

func TestRegistrarsReturnsCopy(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(func(router Router) {})

	registrars := Registrars()
	registrars[0] = func(router Router) {
		router.POST("/mutated", func(w http.ResponseWriter, r *http.Request) {})
	}

	router := &recordingRouter{}
	for _, registrar := range Registrars() {
		registrar(router)
	}
	assert.Empty(t, router.routes)
}

func TestParamReadsRegisteredPathParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/bk_plugin/plugin_api/tasks/42", nil)
	req = WithParams(req, map[string]string{"id": "42"})

	assert.Equal(t, "42", Param(req, "id"))
	assert.Empty(t, Param(req, "missing"))
}
