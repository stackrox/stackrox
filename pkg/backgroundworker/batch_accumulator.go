package backgroundworker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
)

const workerKindAccumulator = "accumulator"

// BatchAccumulator collects items via Add and periodically flushes them
// in batch. This is the standard micro-batching pattern: flush fires on
// a timer, when MaxBatchSize is reached, or on Stop (final drain).
//
// Configured via exported fields. The default Collector is SliceCollector
// (simple append). Use SetCollector or MapCollector for deduplication.
type BatchAccumulator[T any] struct {
	// Name identifies the accumulator for logging, metrics, and status.
	Name string
	// FlushInterval is how often accumulated items are flushed.
	FlushInterval time.Duration
	// Flush is called with the accumulated items. Errors are counted
	// but do not stop the accumulator.
	Flush func(ctx context.Context, items []T) error

	// MaxBatchSize triggers an early flush when the batch reaches this
	// size. Zero means no size limit (timer-only).
	MaxBatchSize int
	// Collector is the accumulation strategy. Defaults to SliceCollector
	// if nil. Set to SetCollector or MapCollector for deduplication.
	Collector Collector[T]

	// EagerFlushRateLimit, when > 0, causes each Add to signal the flush
	// loop (rate-limited to at most one signal per this duration). Use for
	// components that need sub-interval flush responsiveness without
	// flushing on every single Add.
	EagerFlushRateLimit time.Duration

	mu              sync.Mutex
	flushSignal     chan struct{}
	stopper         concurrency.Stopper
	cancel          context.CancelFunc
	state           string
	flushCount      int64
	errorCount      int64
	lastFlushAt     time.Time
	lastFlushDur    time.Duration
	lastEagerSignal time.Time
	started         atomic.Bool
}

// Add appends an item to the batch. Thread-safe. If MaxBatchSize is set
// and the batch has reached it, signals the loop to flush immediately.
// If EagerFlushRateLimit is set, also signals a rate-limited flush.
func (b *BatchAccumulator[T]) Add(item T) {
	b.mu.Lock()
	c := b.collector()
	c.Add(item)
	size := c.Len()
	shouldSignal := b.MaxBatchSize > 0 && size >= b.MaxBatchSize
	if !shouldSignal && b.EagerFlushRateLimit > 0 {
		if time.Since(b.lastEagerSignal) >= b.EagerFlushRateLimit {
			shouldSignal = true
			b.lastEagerSignal = time.Now()
		}
	}
	b.mu.Unlock()

	if shouldSignal {
		select {
		case b.flushSignal <- struct{}{}:
		default:
		}
	}
}

// Start begins the flush loop in a new goroutine. Idempotent.
func (b *BatchAccumulator[T]) Start(ctx context.Context) {
	if !b.started.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	stopper := concurrency.NewStopper()

	b.mu.Lock()
	b.stopper = stopper
	b.cancel = cancel
	b.state = stateRunning
	b.flushSignal = make(chan struct{}, 1)
	b.mu.Unlock()

	go b.loop(ctx, stopper)
}

// Stop signals the loop to exit, performs a final flush of any remaining
// items, and blocks until complete.
func (b *BatchAccumulator[T]) Stop() {
	b.mu.Lock()
	stopper := b.stopper
	cancel := b.cancel
	b.mu.Unlock()

	if stopper == nil {
		return
	}

	if cancel != nil {
		cancel()
	}
	stopper.Client().Stop()
	_ = stopper.Client().Stopped().Wait()

	b.mu.Lock()
	b.state = stateStopped
	b.mu.Unlock()
}

// Status returns a snapshot of the accumulator's runtime state.
func (b *BatchAccumulator[T]) Status() WorkerStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.state
	if state == "" {
		state = stateIdle
	}

	pending := 0
	if b.Collector != nil {
		pending = b.Collector.Len()
	}

	return WorkerStatus{
		Name:        b.Name,
		Kind:        workerKindAccumulator,
		State:       state,
		Interval:    b.FlushInterval,
		LastRunTime: b.lastFlushAt,
		LastRunDur:  b.lastFlushDur,
		RunCount:    b.flushCount,
		ErrorCount:  b.errorCount,
		InFlight:    int64(pending),
	}
}

func (b *BatchAccumulator[T]) loop(ctx context.Context, stopper concurrency.Stopper) {
	defer stopper.Flow().ReportStopped()
	defer b.finalFlush()

	ticker := time.NewTicker(b.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopper.Flow().StopRequested():
			return
		case <-ticker.C:
			b.flushOnce(ctx)
		case <-b.flushSignal:
			b.flushOnce(ctx)
		}
	}
}

func (b *BatchAccumulator[T]) finalFlush() {
	b.flushOnce(context.Background())
}

func (b *BatchAccumulator[T]) flushOnce(ctx context.Context) {
	b.mu.Lock()
	items := b.collector().DrainAll()
	b.mu.Unlock()

	if len(items) == 0 {
		return
	}

	labels := prometheus.Labels{"name": b.Name, "kind": workerKindAccumulator}
	runningGauge.With(labels).Set(1)
	defer runningGauge.With(labels).Set(0)

	start := time.Now()
	err := b.Flush(ctx, items)
	dur := time.Since(start)

	runsTotal.With(labels).Inc()
	runDuration.With(labels).Observe(dur.Seconds())
	if err != nil {
		errorsTotal.With(labels).Inc()
	}

	b.mu.Lock()
	b.lastFlushAt = start
	b.lastFlushDur = dur
	b.flushCount++
	if err != nil {
		b.errorCount++
	}
	b.mu.Unlock()
}

func (b *BatchAccumulator[T]) collector() Collector[T] {
	if b.Collector == nil {
		b.Collector = &SliceCollector[T]{}
	}
	return b.Collector
}
