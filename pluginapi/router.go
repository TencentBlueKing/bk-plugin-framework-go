// Package pluginapi provides framework-level registration for plugin custom APIs.
package pluginapi

import (
	"net/http"
	"sync"
)

// Router is the framework-owned route registration abstraction for plugin APIs.
type Router interface {
	Handle(method string, path string, handler http.HandlerFunc)
	GET(path string, handler http.HandlerFunc)
	POST(path string, handler http.HandlerFunc)
}

// Registrar registers one or more plugin API routes.
type Registrar func(Router)

var (
	mu         sync.RWMutex
	registrars []Registrar
)

// Register stores a plugin API registrar. Runtime implementations decide how to
// expose these routes over their HTTP framework.
func Register(registrar Registrar) {
	mu.Lock()
	defer mu.Unlock()
	registrars = append(registrars, registrar)
}

// Registrars returns a copy of registered plugin API registrars.
func Registrars() []Registrar {
	mu.RLock()
	defer mu.RUnlock()
	copied := make([]Registrar, len(registrars))
	copy(copied, registrars)
	return copied
}

// ResetForTest clears registered plugin APIs.
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	registrars = nil
}
