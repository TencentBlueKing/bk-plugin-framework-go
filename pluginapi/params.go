package pluginapi

import (
	"context"
	"net/http"
)

type paramsContextKey struct{}

// WithParams returns a request carrying path params supplied by the runtime.
func WithParams(r *http.Request, params map[string]string) *http.Request {
	copied := make(map[string]string, len(params))
	for key, value := range params {
		copied[key] = value
	}
	return r.WithContext(context.WithValue(r.Context(), paramsContextKey{}, copied))
}

// Param returns a path param attached by the runtime.
func Param(r *http.Request, name string) string {
	params, ok := r.Context().Value(paramsContextKey{}).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}
