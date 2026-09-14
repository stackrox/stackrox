package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	for name, ready := range map[string]bool{"waiting for ownership": false, "owns scheduler": true} {
		t.Run(name, func(t *testing.T) {
			handler := healthHandler(func() bool { return ready })
			live := httptest.NewRecorder()
			handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			assert.Equal(t, http.StatusOK, live.Code)
			probe := httptest.NewRecorder()
			handler.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			expected := http.StatusServiceUnavailable
			if ready {
				expected = http.StatusOK
			}
			assert.Equal(t, expected, probe.Code)
		})
	}
}

func TestStopBackgroundTasks(t *testing.T) {
	schedulerStopped := make(chan struct{})
	pruningStopped := make(chan struct{})
	done := make(chan struct{})
	go func() {
		stopBackgroundTasks(
			func() { <-schedulerStopped; close(pruningStopped) },
			func() { close(schedulerStopped) },
		)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pruning blocked scheduler shutdown")
	}
	select {
	case <-pruningStopped:
	default:
		t.Fatal("shutdown did not wait for pruning")
	}
}
