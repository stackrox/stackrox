// Package backgroundworker provides standardized archetypes for Central's
// background workers. It owns goroutine lifecycle, stop signaling, metrics,
// and status tracking so individual workers do not need to reimplement them.
package backgroundworker

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	workerKindPeriodic = "periodic"

	stateIdle    = "idle"
	stateRunning = "running"
	stateStopped = "stopped"
)

// PeriodicWorker runs a function on a fixed interval, optionally also on
// start and in response to an external short-circuit signal.
//
// Configured via exported fields (no constructor). Start begins the loop;
// Stop cancels the context and waits for exit. Start and Stop are both
// idempotent — calling Start twice is a no-op, calling Stop twice is safe.
type PeriodicWorker struct {
	// Name identifies the worker for logging, metrics, and status reporting.
	Name string
	// Interval is how often Run is invoked. Zero means signal-only mode:
	// Run fires only via RunOnStart and/or ShortCircuit.
	Interval time.Duration
	// Run is invoked on each tick/signal. Errors are counted but do not
	// stop the worker. Return ErrSkipped to indicate the run was
	// intentionally skipped (not counted in metrics).
	Run func(ctx context.Context) error

	// RunOnStart causes Run (or OnStart, if set) to fire once immediately
	// when Start is called.
	RunOnStart bool
	// ShortCircuit triggers an immediate Run (or OnShortCircuit, if set)
	// when a value is received. Must be a resettable channel (buffered
	// chan struct{}, or SignalChannel.C()). Do NOT use
	// concurrency.Signal.Done() — it stays closed and causes spin.
	ShortCircuit <-chan struct{}
	// ShortCircuitRateLimit is the minimum interval between short-circuit
	// triggered runs. Signals arriving before the limit are dropped.
	ShortCircuitRateLimit time.Duration
	// JitterPct adds ±jitter to each tick interval to prevent thundering
	// herd. 0.1 means ±10%. Only applies when Interval > 0.
	JitterPct float64
	// OnStop is called after the loop exits but before Stop returns.
	// Receives a background context (the main context is already
	// cancelled). Use for final flushes or cleanup.
	OnStop func(ctx context.Context) error

	// OnShortCircuit, if set, is called instead of Run when a short-circuit
	// signal fires. Use when short-circuit triggers need different logic
	// than periodic ticks (e.g., different fetch options).
	OnShortCircuit func(ctx context.Context) error
	// OnStart, if set, is called instead of Run for the initial RunOnStart
	// invocation. Use when the first run needs different logic (e.g.,
	// loading cached state before entering the periodic loop).
	OnStart func(ctx context.Context) error

	// SkipIfRunning causes ticks/signals to be silently dropped while a
	// previous Run is still executing. Without this, the single-goroutine
	// loop naturally serializes runs (the next tick waits). This field is
	// useful when Run is also callable from external code paths and you
	// want to prevent concurrent execution.
	SkipIfRunning bool

	mu          sync.Mutex
	stopper     concurrency.Stopper
	cancel      context.CancelFunc
	state       string
	lastRunTime time.Time
	lastRunDur  time.Duration
	runCount    int64
	errorCount  int64
	started     atomic.Bool
	running     atomic.Bool
}

// WorkerStatus is the runtime state of a worker, queryable for debug endpoints.
type WorkerStatus struct {
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	State       string         `json:"state"`
	Interval    time.Duration  `json:"interval"`
	LastRunTime time.Time      `json:"last_run_time"`
	LastRunDur  time.Duration  `json:"last_run_duration"`
	RunCount    int64          `json:"run_count"`
	ErrorCount  int64          `json:"error_count"`
	InFlight    int64          `json:"in_flight,omitempty"`
	Concurrency int            `json:"concurrency,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// Start begins the worker's loop in a new goroutine. Returns immediately.
// Idempotent — calling Start on an already-started worker is a no-op.
func (w *PeriodicWorker) Start(ctx context.Context) {
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

// Stop cancels the context, signals the loop to exit, and blocks until
// it has. Calls OnStop (if set) after the loop exits. No-op if never
// started or already stopped.
func (w *PeriodicWorker) Stop() {
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
	w.state = stateStopped
	w.mu.Unlock()
}

// Status returns a snapshot of the worker's current runtime state.
func (w *PeriodicWorker) Status() WorkerStatus {
	w.mu.Lock()
	defer w.mu.Unlock()

	state := w.state
	if state == "" {
		state = stateIdle
	}

	return WorkerStatus{
		Name:        w.Name,
		Kind:        workerKindPeriodic,
		State:       state,
		Interval:    w.Interval,
		LastRunTime: w.lastRunTime,
		LastRunDur:  w.lastRunDur,
		RunCount:    w.runCount,
		ErrorCount:  w.errorCount,
	}
}

func (w *PeriodicWorker) loop(ctx context.Context, stopper concurrency.Stopper) {
	defer stopper.Flow().ReportStopped()
	defer w.runOnStop()

	if w.RunOnStart {
		if w.OnStart != nil {
			w.runFunc(ctx, w.OnStart)
		} else {
			w.runFunc(ctx, w.Run)
		}
	}

	if w.Interval <= 0 && w.ShortCircuit == nil {
		return
	}

	var lastShortCircuit time.Time

	for {
		timerC := w.nextTick()

		select {
		case <-stopper.Flow().StopRequested():
			return
		case <-timerC:
			w.runFunc(ctx, w.Run)
		case <-w.ShortCircuit:
			if w.ShortCircuitRateLimit > 0 && time.Since(lastShortCircuit) < w.ShortCircuitRateLimit {
				continue
			}
			lastShortCircuit = time.Now()
			if w.OnShortCircuit != nil {
				w.runFunc(ctx, w.OnShortCircuit)
			} else {
				w.runFunc(ctx, w.Run)
			}
		}
	}
}

// nextTick returns a channel that fires after the next interval, with
// jitter applied if JitterPct > 0. Returns nil if Interval <= 0.
func (w *PeriodicWorker) nextTick() <-chan time.Time {
	if w.Interval <= 0 {
		return nil
	}
	d := w.Interval
	if w.JitterPct > 0 {
		jitter := float64(d) * w.JitterPct * (2*rand.Float64() - 1)
		d = time.Duration(float64(d) + jitter)
		if d <= 0 {
			d = 1
		}
	}
	return time.After(d)
}

func (w *PeriodicWorker) runOnStop() {
	if w.OnStop == nil {
		return
	}
	_ = w.OnStop(context.Background())
}

func (w *PeriodicWorker) runFunc(ctx context.Context, fn func(ctx context.Context) error) {
	if w.SkipIfRunning && !w.running.CompareAndSwap(false, true) {
		return
	}
	if w.SkipIfRunning {
		defer w.running.Store(false)
	}

	labels := prometheus.Labels{"name": w.Name, "kind": workerKindPeriodic}
	runningGauge.With(labels).Set(1)
	defer runningGauge.With(labels).Set(0)

	start := time.Now()
	err := fn(ctx)
	dur := time.Since(start)

	if errors.Is(err, ErrSkipped) {
		return
	}

	runsTotal.With(labels).Inc()
	runDuration.With(labels).Observe(dur.Seconds())
	if err != nil {
		errorsTotal.With(labels).Inc()
	}

	w.mu.Lock()
	w.lastRunTime = start
	w.lastRunDur = dur
	w.runCount++
	if err != nil {
		w.errorCount++
	}
	w.mu.Unlock()
}
