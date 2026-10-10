package backgroundworker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicWorker_RunsOnInterval(t *testing.T) {
	var count atomic.Int32
	w := &PeriodicWorker{
		Name:     "test-periodic",
		Interval: 50 * time.Millisecond,
		Run: func(_ context.Context) error {
			count.Add(1)
			return nil
		},
	}
	w.Start(context.Background())
	time.Sleep(180 * time.Millisecond)
	w.Stop()
	// Should have run 3 times (at 50ms, 100ms, 150ms)
	assert.GreaterOrEqual(t, count.Load(), int32(3))
}

func TestPeriodicWorker_RunOnStart(t *testing.T) {
	var count atomic.Int32
	w := &PeriodicWorker{
		Name:       "test-run-on-start",
		Interval:   time.Hour,
		RunOnStart: true,
		Run: func(_ context.Context) error {
			count.Add(1)
			return nil
		},
	}
	w.Start(context.Background())
	time.Sleep(20 * time.Millisecond)
	w.Stop()
	assert.Equal(t, int32(1), count.Load())
}

func TestPeriodicWorker_ShortCircuit(t *testing.T) {
	var count atomic.Int32
	sig := make(chan struct{}, 1)
	w := &PeriodicWorker{
		Name:         "test-short-circuit",
		Interval:     time.Hour,
		ShortCircuit: sig,
		Run: func(_ context.Context) error {
			count.Add(1)
			return nil
		},
	}
	w.Start(context.Background())
	sig <- struct{}{}
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load())
	w.Stop()
}

func TestPeriodicWorker_StatusTracking(t *testing.T) {
	w := &PeriodicWorker{
		Name:       "test-status",
		Interval:   50 * time.Millisecond,
		RunOnStart: true,
		Run: func(_ context.Context) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	}
	w.Start(context.Background())
	time.Sleep(30 * time.Millisecond)
	s := w.Status()
	assert.Equal(t, "test-status", s.Name)
	assert.Equal(t, "periodic", s.Kind)
	assert.Equal(t, "running", s.State)
	assert.GreaterOrEqual(t, s.RunCount, int64(1))
	assert.False(t, s.LastRunTime.IsZero())
	assert.Greater(t, s.LastRunDur, time.Duration(0))
	w.Stop()
	assert.Equal(t, "stopped", w.Status().State)
}

func TestPeriodicWorker_ErrorCounting(t *testing.T) {
	callNum := atomic.Int32{}
	w := &PeriodicWorker{
		Name:       "test-errors",
		Interval:   50 * time.Millisecond,
		RunOnStart: true,
		Run: func(_ context.Context) error {
			n := callNum.Add(1)
			if n%2 == 1 {
				return fmt.Errorf("odd call %d", n)
			}
			return nil
		},
	}
	w.Start(context.Background())
	time.Sleep(180 * time.Millisecond)
	w.Stop()
	s := w.Status()
	require.GreaterOrEqual(t, s.RunCount, int64(3))
	assert.Greater(t, s.ErrorCount, int64(0))
	assert.Less(t, s.ErrorCount, s.RunCount)
}

func TestPeriodicWorker_StatusBeforeStart(t *testing.T) {
	w := &PeriodicWorker{
		Name:     "test-not-started",
		Interval: 50 * time.Millisecond,
		Run: func(_ context.Context) error {
			return nil
		},
	}
	s := w.Status()
	assert.Equal(t, "test-not-started", s.Name)
	assert.Equal(t, "periodic", s.Kind)
	assert.Equal(t, "idle", s.State)
	assert.Equal(t, 50*time.Millisecond, s.Interval)
	assert.Zero(t, s.RunCount)
	assert.Zero(t, s.ErrorCount)
	assert.True(t, s.LastRunTime.IsZero())
}

func TestPeriodicWorker_StopBeforeStartIsNoOp(t *testing.T) {
	w := &PeriodicWorker{
		Name:     "test-stop-before-start",
		Interval: 50 * time.Millisecond,
		Run: func(_ context.Context) error {
			return nil
		},
	}
	assert.NotPanics(t, func() {
		w.Stop()
	})
	assert.Equal(t, "idle", w.Status().State)
}

func TestPeriodicWorker_ShortCircuitRateLimit(t *testing.T) {
	var count atomic.Int32
	sig := make(chan struct{}, 2)
	w := &PeriodicWorker{
		Name:                  "test-short-circuit-rate-limit",
		Interval:              time.Hour,
		ShortCircuit:          sig,
		ShortCircuitRateLimit: 100 * time.Millisecond,
		Run: func(_ context.Context) error {
			count.Add(1)
			return nil
		},
	}
	w.Start(context.Background())
	sig <- struct{}{}
	time.Sleep(10 * time.Millisecond)
	sig <- struct{}{}
	time.Sleep(20 * time.Millisecond)
	w.Stop()
	// The second signal arrived within the rate limit window and should have
	// been dropped, so Run should have executed only once.
	assert.Equal(t, int32(1), count.Load())
}
