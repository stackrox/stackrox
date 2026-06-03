package healthz

import (
	"net/http"
)

// Handler provides liveness and readiness health check endpoints.
type Handler struct {
	isReady func() bool
}

// NewHandler creates a new health check handler.
// isReady is called to determine readiness status; nil means always ready.
func NewHandler(isReady func() bool) *Handler {
	return &Handler{isReady: isReady}
}

// ServeHTTP implements http.Handler for health check routes.
// Supports:
// - /healthz/live - always returns 200 OK
// - /healthz/ready - returns 200 OK if ready, 503 Service Unavailable otherwise
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/healthz/live":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	case "/healthz/ready":
		if h.isReady == nil || h.isReady() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Not Ready"))
		}
	default:
		http.NotFound(w, r)
	}
}
