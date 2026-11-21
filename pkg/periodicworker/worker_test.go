package periodicworker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Validation(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		_, err := New(Config{
			Name:     "",
			Interval: time.Second,
			Task:     func(ctx context.Context) error { return nil },
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Name")
	})

	t.Run("zero interval", func(t *testing.T) {
		_, err := New(Config{
			Name:     "test",
			Interval: 0,
			Task:     func(ctx context.Context) error { return nil },
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Interval")
	})

	t.Run("nil task", func(t *testing.T) {
		_, err := New(Config{
			Name:     "test",
			Interval: time.Second,
			Task:     nil,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Task")
	})

	t.Run("valid config", func(t *testing.T) {
		w, err := New(Config{
			Name:     "test",
			Interval: time.Second,
			Task:     func(ctx context.Context) error { return nil },
		})
		assert.NoError(t, err)
		assert.NotNil(t, w)
	})
}

func TestMustNew_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustNew(Config{
			Name:     "",
			Interval: time.Second,
			Task:     func(ctx context.Context) error { return nil },
		})
	})
}

func TestWorker_RunsOnInterval(t *testing.T) {
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:     "test_interval",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	defer w.Stop()

	// Wait for at least 3 runs
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 3
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_RunOnStart(t *testing.T) {
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:       "test_run_on_start",
		Interval:   100 * time.Millisecond,
		RunOnStart: true,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	defer w.Stop()

	// Should run immediately
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 1
	}, 50*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_GracefulStop(t *testing.T) {
	started := concurrency.NewSignal()
	blockTask := concurrency.NewSignal()

	w := MustNew(Config{
		Name:     "test_graceful_stop",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			started.Signal()
			blockTask.Wait()
			return nil
		},
	})

	w.Start()

	// Wait for task to start
	started.Wait()

	// Stop should block until task completes
	stopDone := concurrency.NewSignal()
	go func() {
		w.Stop()
		stopDone.Signal()
	}()

	// Give it a moment, Stop should still be blocking
	time.Sleep(20 * time.Millisecond)
	assert.False(t, stopDone.IsDone())

	// Unblock the task
	blockTask.Signal()

	// Now Stop should complete
	assert.Eventually(t, func() bool {
		return stopDone.IsDone()
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_Kick(t *testing.T) {
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:     "test_kick",
		Interval: 1 * time.Second, // Long interval so kicks are observable
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	defer w.Stop()

	// Kick multiple times
	w.Kick()
	w.Kick()
	w.Kick()

	// Should run from kicks
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 1
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_ExternalKick(t *testing.T) {
	var mu sync.Mutex
	runCount := 0
	kickSignal := concurrency.NewSignal()

	w := MustNew(Config{
		Name:     "test_external_kick",
		Interval: 1 * time.Second,
		Kick:     &kickSignal,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	defer w.Stop()

	// Kick via external signal
	kickSignal.Signal()

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 1
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_ErrorHandling(t *testing.T) {
	testErr := errors.New("test error")
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:     "test_error",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return testErr
		},
	})

	w.Start()
	defer w.Stop()

	// Worker should continue despite errors
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 3
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_PanicRecovery(t *testing.T) {
	var mu sync.Mutex
	runCount := 0
	shouldPanic := true

	w := MustNew(Config{
		Name:     "test_panic",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			panicNow := shouldPanic
			shouldPanic = false
			mu.Unlock()

			if panicNow {
				panic("test panic")
			}
			return nil
		},
	})

	w.Start()
	defer w.Stop()

	// First run should panic, but worker should survive and continue
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return runCount >= 2
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_Timeout(t *testing.T) {
	var mu sync.Mutex
	timeoutOccurred := false

	w := MustNew(Config{
		Name:     "test_timeout",
		Interval: 10 * time.Millisecond,
		Timeout:  5 * time.Millisecond,
		Task: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				mu.Lock()
				timeoutOccurred = true
				mu.Unlock()
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		},
	})

	w.Start()
	defer w.Stop()

	// Timeout should occur
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return timeoutOccurred
	}, 100*time.Millisecond, 2*time.Millisecond)
}

func TestWorker_ContextCancelledOnStop(t *testing.T) {
	started := concurrency.NewSignal()
	var mu sync.Mutex
	ctxCancelled := false

	w := MustNew(Config{
		Name:     "test_ctx_cancel",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			started.Signal()
			<-ctx.Done()
			mu.Lock()
			ctxCancelled = true
			mu.Unlock()
			return ctx.Err()
		},
	})

	w.Start()
	started.Wait()
	w.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, ctxCancelled)
}

func TestWorker_LastSuccess(t *testing.T) {
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:     "test_last_success",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			shouldFail := runCount == 1
			mu.Unlock()

			if shouldFail {
				return errors.New("first run fails")
			}
			return nil
		},
	})

	// Initial LastSuccess should be zero
	assert.True(t, w.LastSuccess().IsZero())

	w.Start()
	defer w.Stop()

	// Wait for successful run (second run)
	assert.Eventually(t, func() bool {
		return !w.LastSuccess().IsZero()
	}, 100*time.Millisecond, 2*time.Millisecond)

	lastSuccess := w.LastSuccess()
	assert.False(t, lastSuccess.IsZero())
	assert.True(t, time.Since(lastSuccess) < 100*time.Millisecond)
}

func TestWorker_StartIsIdempotent(t *testing.T) {
	var mu sync.Mutex
	runCount := 0

	w := MustNew(Config{
		Name:     "test_idempotent",
		Interval: 10 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	w.Start()
	w.Start()
	defer w.Stop()

	// Should only have one goroutine running
	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	count := runCount
	mu.Unlock()

	// Count should be reasonable for one worker (not 3x)
	assert.True(t, count >= 1 && count <= 5, "runCount=%d", count)
}

func TestWorker_StoppedSignal(t *testing.T) {
	w := MustNew(Config{
		Name:     "test_stopped_signal",
		Interval: 10 * time.Millisecond,
		Task:     func(ctx context.Context) error { return nil },
	})

	w.Start()

	stoppedSig := w.Stopped()
	require.NotNil(t, stoppedSig)
	assert.False(t, concurrency.IsDone(stoppedSig))

	w.Stop()

	assert.True(t, concurrency.IsDone(stoppedSig))
}

func TestWorker_NoOverlappingRuns(t *testing.T) {
	inProgress := 0
	var mu sync.Mutex
	maxConcurrent := 0

	w := MustNew(Config{
		Name:     "test_no_overlap",
		Interval: 5 * time.Millisecond,
		Task: func(ctx context.Context) error {
			mu.Lock()
			inProgress++
			if inProgress > maxConcurrent {
				maxConcurrent = inProgress
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			inProgress--
			mu.Unlock()
			return nil
		},
	})

	w.Start()
	time.Sleep(50 * time.Millisecond)
	w.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, maxConcurrent, "should never have overlapping runs")
}
