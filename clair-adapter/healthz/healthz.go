package healthz

import (
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
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
		h.live(w, r)
	case "/healthz/ready":
		h.ready(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) ready(w http.ResponseWriter, _ *http.Request) {
	if h.isReady == nil || h.isReady() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Not Ready"))
	}
}

// CustomRoutes returns the health check routes for pkg/grpc.API integration.
func (h *Handler) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/healthz/live",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(h.live),
			Compression:   false,
		},
		{
			Route:         "/healthz/ready",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(h.ready),
			Compression:   false,
		},
	}
}
