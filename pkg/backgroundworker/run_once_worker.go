package backgroundworker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	workerKindOnce = "once"

	stateCompleted = "completed"
	stateFailed    = "failed"
	stateRetrying  = "retrying"

	defaultRetryInterval    = time.Minute
	defaultBackoffMultipler = 2.0
	defaultMaxRetryInterval = 10 * time.Minute
)

// RunOnceWorker runs a function to completion with retry and exponential
// backoff. This is the standard pattern for startup tasks and one-time
// migrations that need lifecycle management but are not periodic.
//
// The worker transitions through states: idle → running → retrying →
// completed (on success) or failed (on exhausted attempts / permanent
// error). Stop cancels at any point.
type RunOnceWorker struct {
	// Name identifies the worker for logging, metrics, and status.
	Name string
	// Run is the function to execute. Return nil for success.
	Run func(ctx context.Context) error

	// RetryInterval is the base delay between retries. Default: 1m.
	RetryInterval time.Duration
	// BackoffMultiplier scales RetryInterval after each failed attempt.
	// Default: 2.0 (exponential backoff).
	BackoffMultiplier float64
	// MaxRetryInterval caps the backoff delay. Default: 10m.
	MaxRetryInterval time.Duration
	// MaxAttempts is the maximum number of attempts. Zero means unlimited.
	MaxAttempts int
	// ShouldRetry classifies errors. If non-nil and returns false, the
	// worker transitions to "failed" without further retries. If nil,
	// all errors are retried.
	ShouldRetry func(error) bool

	mu          sync.Mutex
	stopper     concurrency.Stopper
	cancel      context.CancelFunc
	state       string
	attempts    int64
	lastErr     error
	lastRunTime time.Time
	lastRunDur  time.Duration
	started     atomic.Bool
}

// Start begins execution in a new goroutine. Idempotent.
func (w *RunOnceWorker) Start(ctx context.Context) {
	if !w.started.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	stopper := concurrency.NewStopper()

	w.mu.Lock()
	w.stopper = stopper
	w.cancel = cancel
	w.state = stateRunning
	w.mu.Unlock()

	go w.loop(ctx, stopper)
}

// Stop cancels execution and blocks until the worker exits.
func (w *RunOnceWorker) Stop() {
	w.mu.Lock()
	stopper := w.stopper
	cancel := w.cancel
	w.mu.Unlock()

	if stopper == nil {
		return
	}

	if cancel != nil {
		cancel()
	}
	stopper.Client().Stop()
	_ = stopper.Client().Stopped().Wait()

	w.mu.Lock()
	if w.state == stateRunning || w.state == stateRetrying {
		w.state = stateStopped
	}
	w.mu.Unlock()
}

// Status returns a snapshot of the worker's runtime state.
func (w *RunOnceWorker) Status() WorkerStatus {
	w.mu.Lock()
	defer w.mu.Unlock()

	state := w.state
	if state == "" {
		state = stateIdle
	}

	return WorkerStatus{
		Name:        w.Name,
		Kind:        workerKindOnce,
		State:       state,
		LastRunTime: w.lastRunTime,
		LastRunDur:  w.lastRunDur,
		RunCount:    w.attempts,
		ErrorCount:  w.attempts - boolToInt64(w.state == stateCompleted),
	}
}

func (w *RunOnceWorker) loop(ctx context.Context, stopper concurrency.Stopper) {
	defer stopper.Flow().ReportStopped()

	labels := prometheus.Labels{"name": w.Name, "kind": workerKindOnce}

	for attempt := 0; ; attempt++ {
		runningGauge.With(labels).Set(1)
		start := time.Now()
		err := w.Run(ctx)
		dur := time.Since(start)
		runningGauge.With(labels).Set(0)

		runsTotal.With(labels).Inc()
		runDuration.With(labels).Observe(dur.Seconds())

		w.mu.Lock()
		w.attempts++
		w.lastRunTime = start
		w.lastRunDur = dur
		w.lastErr = err
		w.mu.Unlock()

		if err == nil {
			w.mu.Lock()
			w.state = stateCompleted
			w.mu.Unlock()
			return
		}

		errorsTotal.With(labels).Inc()

		if w.MaxAttempts > 0 && attempt+1 >= w.MaxAttempts {
			w.mu.Lock()
			w.state = stateFailed
			w.mu.Unlock()
			return
		}

		if w.ShouldRetry != nil && !w.ShouldRetry(err) {
			w.mu.Lock()
			w.state = stateFailed
			w.mu.Unlock()
			return
		}

		w.mu.Lock()
		w.state = stateRetrying
		w.mu.Unlock()

		delay := w.backoffDelay(attempt)
		select {
		case <-ctx.Done():
			return
		case <-stopper.Flow().StopRequested():
			return
		case <-time.After(delay):
		}
	}
}

func (w *RunOnceWorker) backoffDelay(attempt int) time.Duration {
	base := w.RetryInterval
	if base <= 0 {
		base = defaultRetryInterval
	}
	multiplier := w.BackoffMultiplier
	if multiplier < 1 {
		multiplier = defaultBackoffMultipler
	}
	maxInterval := w.MaxRetryInterval
	if maxInterval <= 0 {
		maxInterval = defaultMaxRetryInterval
	}

	d := float64(base)
	for range attempt {
		d *= multiplier
		if d > float64(maxInterval) {
			d = float64(maxInterval)
			break
		}
	}
	return time.Duration(d)
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
