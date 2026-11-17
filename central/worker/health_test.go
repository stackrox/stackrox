package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthDuringStartupAndShutdown(t *testing.T) {
	var ready atomic.Bool
	handler := newHealthHandler(&ready)
	check := func(path string, status int) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, status, response.Code, path)
	}

	// A long migration must not fail liveness or allow reporting work to start.
	check("/healthz", http.StatusOK)
	check("/readyz", http.StatusServiceUnavailable)
	ready.Store(true)
	check("/healthz", http.StatusOK)
	check("/readyz", http.StatusOK)
	ready.Store(false)
	check("/healthz", http.StatusOK)
	check("/readyz", http.StatusServiceUnavailable)
}
