// Package backgroundworker provides standardized archetypes for Central's
// background workers, such as periodic polling loops. It is intended as a
// shared building block: it owns goroutine lifecycle, stop signaling, and
// status tracking so individual workers do not need to reimplement them.
package backgroundworker

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	// workerKindPeriodic identifies workers driven by PeriodicWorker in WorkerStatus.Kind.
	workerKindPeriodic = "periodic"

	// stateIdle is the state of a PeriodicWorker that has been constructed but not yet started.
	stateIdle = "idle"
	// stateRunning is the state of a PeriodicWorker whose loop goroutine is active.
	stateRunning = "running"
	// stateStopped is the state of a PeriodicWorker that has been stopped.
	stateStopped = "stopped"
)

// PeriodicWorker runs a function on a fixed interval, optionally also on
// start and in response to an external short-circuit signal. It is
// configured via its exported fields (there is no constructor) and started
// with Start; callers must call Stop to release the underlying goroutine.
//
// A PeriodicWorker must not be started more than once, and its exported
// configuration fields (Name, Interval, Run, RunOnStart, ShortCircuit,
// ShortCircuitRateLimit) must not be modified after Start is called.
type PeriodicWorker struct {
	// Name identifies the worker, e.g. for logging and status reporting.
	Name string
	// Interval is how often Run is invoked. If zero, Run is only invoked via
	// RunOnStart and/or ShortCircuit.
	Interval time.Duration
	// Run is invoked on each tick (and possibly on start / short-circuit).
	// Its returned error is counted in WorkerStatus.ErrorCount but does not
	// stop the worker.
	Run func(ctx context.Context) error

	// RunOnStart, if true, causes Run to be invoked once immediately when
	// Start is called, in addition to the regular interval/short-circuit
	// triggers.
	RunOnStart bool
	// ShortCircuit, if non-nil, is an additional trigger for Run: whenever a
	// value is received on this channel, Run is invoked (subject to
	// ShortCircuitRateLimit).
	ShortCircuit <-chan struct{}
	// ShortCircuitRateLimit, if positive, is the minimum time that must
	// elapse between two Run invocations triggered by ShortCircuit. Signals
	// received before the rate limit has elapsed are dropped.
	ShortCircuitRateLimit time.Duration

	mu          sync.Mutex
	stopper     concurrency.Stopper
	cancel      context.CancelFunc
	state       string
	lastRunTime time.Time
	lastRunDur  time.Duration
	runCount    int64
	errorCount  int64
}

// WorkerStatus is the runtime state of a worker, queryable for debug endpoints.
type WorkerStatus struct {
	Name        string        `json:"name"`
	Kind        string        `json:"kind"`
	State       string        `json:"state"`
	Interval    time.Duration `json:"interval"`
	LastRunTime time.Time     `json:"last_run_time"`
	LastRunDur  time.Duration `json:"last_run_duration"`
	RunCount    int64         `json:"run_count"`
	ErrorCount  int64         `json:"error_count"`
	InFlight    int64         `json:"in_flight,omitempty"`
	Concurrency int           `json:"concurrency,omitempty"`
}

// Start begins running the worker's loop in a new goroutine. It returns
// immediately; the loop runs until Stop is called or ctx is done.
func (w *PeriodicWorker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	stopper := concurrency.NewStopper()

	w.mu.Lock()
	w.stopper = stopper
	w.cancel = cancel
	w.state = stateRunning
	w.mu.Unlock()

	go w.loop(ctx, stopper)
}

// Stop signals the worker's loop to stop and blocks until it has exited.
// It is a no-op if the worker was never started.
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

// Status returns a snapshot of the worker's current runtime state. It is
// safe to call before Start (in which case State is "idle" and the run/error
// counters and timing fields are zero-valued) and concurrently with Start,
// Stop, and the running loop.
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

	if w.RunOnStart {
		w.runOnce(ctx)
	}

	if w.Interval <= 0 && w.ShortCircuit == nil {
		return
	}

	var tickerC <-chan time.Time
	if w.Interval > 0 {
		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()
		tickerC = ticker.C
	}

	var lastShortCircuit time.Time

	for {
		select {
		case <-stopper.Flow().StopRequested():
			return
		case <-tickerC:
			w.runOnce(ctx)
		case <-w.ShortCircuit:
			if w.ShortCircuitRateLimit > 0 && time.Since(lastShortCircuit) < w.ShortCircuitRateLimit {
				continue
			}
			lastShortCircuit = time.Now()
			w.runOnce(ctx)
		}
	}
}

func (w *PeriodicWorker) runOnce(ctx context.Context) {
	labels := prometheus.Labels{"name": w.Name, "kind": workerKindPeriodic}
	runningGauge.With(labels).Set(1)
	defer runningGauge.With(labels).Set(0)

	start := time.Now()
	err := w.Run(ctx)
	dur := time.Since(start)

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
