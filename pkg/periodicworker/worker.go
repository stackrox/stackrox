package periodicworker

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime/debug"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

// Task is one unit of periodic work. It MUST respect ctx cancellation (the worker cancels
// ctx when Stop is requested and when a per-run Timeout elapses). Returning an error is
// expected and non-fatal: it is counted, logged, and the loop continues.
type Task func(ctx context.Context) error

// Config holds the configuration for a periodic worker.
type Config struct {
	// Name is required and unique per process. It becomes the `worker` metric label and the
	// structured-log field, so it must be a stable, low-cardinality identifier
	// (e.g. "networkbaseline_flush", "cve_unsuppress").
	Name string

	// Interval is required: the period between the START of successive runs' scheduling.
	Interval time.Duration

	// Task is required.
	Task Task

	// RunOnStart runs Task once immediately on Start(), before the first interval tick.
	// Many existing loops do this by hand ("immediate first run"); here it is a flag.
	RunOnStart bool

	// Jitter, if > 0, randomizes each interval by ±Jitter to avoid thundering-herd across
	// replicas/workers (scannerdefinitions already hand-rolls this).
	Jitter time.Duration

	// Timeout, if > 0, bounds a single run: Task's ctx is cancelled after Timeout. A run that
	// exceeds it is recorded as outcome="timeout".
	Timeout time.Duration

	// Kick, if set, triggers an out-of-band run as soon as it fires, in addition to the
	// interval. Coalesced: multiple kicks during a run collapse to at most one pending run.
	// This replaces the bespoke "short-circuit signal" in cloudsources, declarativeconfig,
	// reprocessor, and cve/fetcher.
	Kick concurrency.Waitable

	// Logger is optional; defaults to a package logger. The worker always logs with a
	// "worker"=Name field.
	Logger logging.Logger
}

// Worker is a started/stoppable periodic worker. Construct with New, then Start/Stop.
type Worker struct {
	name    string
	cfg     Config
	stopper concurrency.Stopper
	log     logging.Logger

	// Internal kick signal
	kickSignal concurrency.Signal

	// Last success time
	lastSuccessMu sync.Mutex
	lastSuccess   time.Time

	// Started tracking
	startedMu sync.Mutex
	started   bool
}

var (
	packageLogger = logging.LoggerForModule()
)

// New validates cfg and returns a Worker. It does NOT start the goroutine.
func New(cfg Config) (*Worker, error) {
	if cfg.Name == "" {
		return nil, errors.New("Name is required")
	}
	if cfg.Interval <= 0 {
		return nil, errors.New("Interval must be > 0")
	}
	if cfg.Task == nil {
		return nil, errors.New("Task is required")
	}

	log := cfg.Logger
	if log == nil {
		log = packageLogger
	}

	return &Worker{
		name:    cfg.Name,
		cfg:     cfg,
		stopper: concurrency.NewStopper(),
		log:     log,
	}, nil
}

// MustNew is New that panics on invalid config — for package-level/singleton construction.
func MustNew(cfg Config) *Worker {
	w, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("periodicworker.MustNew: %v", err))
	}
	return w
}

// Start launches the loop goroutine. Idempotent; safe to call once.
func (w *Worker) Start() {
	w.startedMu.Lock()
	defer w.startedMu.Unlock()

	if w.started {
		return
	}
	w.started = true

	w.kickSignal = concurrency.NewSignal()
	go w.run()
}

// Stop requests graceful shutdown and BLOCKS until the in-flight run (if any) returns and the
// loop goroutine has exited. This is the behavior ~40% of current loops are missing.
func (w *Worker) Stop() {
	w.stopper.Client().Stop()
	w.stopper.Client().Stopped().Wait()
}

// Stopped exposes the underlying signal for callers that want to wait without blocking in Stop.
func (w *Worker) Stopped() concurrency.ReadOnlyErrorSignal {
	return w.stopper.Client().Stopped()
}

// Kick triggers an out-of-band run now (coalesced). Equivalent to firing Config.Kick.
func (w *Worker) Kick() {
	w.kickSignal.Signal()
}

// LastSuccess returns the time of the last successful run (zero if none). Lets a health check
// assert freshness without the worker depending on any health subsystem.
func (w *Worker) LastSuccess() time.Time {
	w.lastSuccessMu.Lock()
	defer w.lastSuccessMu.Unlock()
	return w.lastSuccess
}

func (w *Worker) setLastSuccess(t time.Time) {
	w.lastSuccessMu.Lock()
	defer w.lastSuccessMu.Unlock()
	w.lastSuccess = t
}

func (w *Worker) run() {
	defer w.stopper.Flow().ReportStopped()

	workerRunning.WithLabelValues(w.name).Set(1)
	defer workerRunning.WithLabelValues(w.name).Set(0)

	w.log.Infow("periodic worker started", "worker", w.name, "interval", w.cfg.Interval)
	defer w.log.Infow("periodic worker stopped", "worker", w.name)

	interval := w.cfg.Interval
	if w.cfg.Jitter > 0 {
		interval = w.applyJitter(interval)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if w.cfg.RunOnStart {
		w.runOnce()
	}

	for {
		// Apply jitter for next interval
		if w.cfg.Jitter > 0 {
			interval = w.applyJitter(w.cfg.Interval)
			ticker.Reset(interval)
		}

		var kick <-chan struct{}
		if w.cfg.Kick != nil {
			kick = w.cfg.Kick.Done()
		}
		internalKick := w.kickSignal.Done()

		select {
		case <-w.stopper.Flow().StopRequested():
			return
		case <-ticker.C:
			w.runOnce()
		case <-kick:
			w.runOnce()
		case <-internalKick:
			w.runOnce()
		}
	}
}

func (w *Worker) applyJitter(interval time.Duration) time.Duration {
	jitterRange := int64(2 * w.cfg.Jitter)
	jitter := time.Duration(rand.Int64N(jitterRange)) - w.cfg.Jitter
	result := interval + jitter
	if result <= 0 {
		return interval
	}
	return result
}

func (w *Worker) runOnce() {
	start := time.Now()
	workerInProgress.WithLabelValues(w.name).Set(1)
	defer workerInProgress.WithLabelValues(w.name).Set(0)

	// Create context from stopper
	ctx := concurrency.AsContext(concurrency.WaitableChan(w.stopper.Flow().StopRequested()))
	if w.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.cfg.Timeout)
		defer cancel()
	}

	outcome := "success"
	var runErr error
	defer func() {
		if r := recover(); r != nil {
			outcome = "panic"
			w.log.Errorw("periodic worker panicked", "worker", w.name, "panic", r, "stack", string(debug.Stack()))
		}

		dur := time.Since(start).Seconds()
		workerRunDuration.WithLabelValues(w.name).Observe(dur)
		workerRunsTotal.WithLabelValues(w.name, outcome).Inc()
		workerLastRun.WithLabelValues(w.name).SetToCurrentTime()

		if outcome == "success" {
			workerLastSuccess.WithLabelValues(w.name).SetToCurrentTime()
			w.setLastSuccess(time.Now())
			workerConsecutiveFailures.WithLabelValues(w.name).Set(0)
		} else {
			workerConsecutiveFailures.WithLabelValues(w.name).Inc()
		}

		if runErr != nil {
			if outcome == "timeout" {
				w.log.Warnw("periodic worker run timed out", "worker", w.name, "timeout", w.cfg.Timeout, "error", runErr)
			} else if outcome != "panic" {
				w.log.Warnw("periodic worker run failed", "worker", w.name, "error", runErr)
			}
		}
	}()

	runErr = w.cfg.Task(ctx)
	if runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) {
			outcome = "timeout"
		} else {
			outcome = "error"
		}
	}
}
