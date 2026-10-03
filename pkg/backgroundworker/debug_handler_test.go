package backgroundworker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugHandler_ReturnsJSON(t *testing.T) {
	r := NewRegistry()
	w := &PeriodicWorker{
		Name:     "debug-test-worker",
		Interval: time.Hour,
		Run:      func(ctx context.Context) error { return nil },
	}
	w.Start(context.Background())
	defer w.Stop()
	r.Register(w)

	req := httptest.NewRequest(http.MethodGet, "/debug/workers", nil)
	rec := httptest.NewRecorder()
	r.DebugHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var statuses []WorkerStatus
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statuses))
	require.Len(t, statuses, 1)
	assert.Equal(t, "debug-test-worker", statuses[0].Name)
	assert.Equal(t, "running", statuses[0].State)
}
