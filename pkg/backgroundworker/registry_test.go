package backgroundworker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterAndList(t *testing.T) {
	r := NewRegistry()
	w := &PeriodicWorker{
		Name:     "test-worker",
		Interval: time.Hour,
		Run:      func(ctx context.Context) error { return nil },
	}
	r.Register(w)
	workers := r.All()
	require.Len(t, workers, 1)
	assert.Equal(t, "test-worker", workers[0].Name)
}
