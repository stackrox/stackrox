package metrics

import (
	"net/http"
)

// HTTPMetrics provides an object which can wrap HTTP handlers and provide a method to get metrics about the wrapped
// HTTP handlers
//
//go:generate mockgen-wrapper
type HTTPMetrics interface {
	WrapHandler(handler http.Handler, path string) http.Handler
	GetMetrics() (map[string]map[int]int64, map[string]map[string]int64)
}

// NewHTTPMetrics returns a new HTTPMetrics object
func NewHTTPMetrics() HTTPMetrics {
	return &httpMetricsImpl{
		allMetrics: make(map[string]*perPathHTTPMetrics),
	}
}
