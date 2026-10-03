package backgroundworker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
)

const workerKindQueue = "queue"

// QueueConsumer drains a typed channel with bounded concurrency.
//
// Configured via exported fields. Start begins draining; Stop cancels
// and waits for in-flight handlers. Not safe to Start twice.
type QueueConsumer[T any] struct {
	// Name identifies the consumer for logging, metrics, and status.
	Name string
	// Source is the channel to drain. When Source is closed, the consumer
	// finishes in-flight handlers and exits.
	Source <-chan T
	// Handle is called for each item received from Source.
	Handle func(ctx context.Context, item T) error
	// Concurrency is the maximum number of concurrent Handle invocations.
	// Defaults to 1 if <= 0.
	Concurrency int

	mu           sync.Mutex
	stopper      concurrency.Stopper
	cancel       context.CancelFunc
	state        string
	itemCount    int64
	errorCount   int64
	inFlight     int64
	lastItemTime time.Time
	lastItemDur  time.Duration
	started      atomic.Bool
}

// Start begins draining Source in a new goroutine. Idempotent.
func (c *QueueConsumer[T]) Start(ctx context.Context) {
	if !c.started.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	stopper := concurrency.NewStopper()

	c.mu.Lock()
	c.stopper = stopper
	c.cancel = cancel
	c.state = stateRunning
	c.mu.Unlock()

	go c.loop(ctx, stopper)
}

// Stop signals the consumer to stop pulling new items and blocks until
// all in-flight handlers have completed.
func (c *QueueConsumer[T]) Stop() {
	c.mu.Lock()
	stopper := c.stopper
	cancel := c.cancel
	c.mu.Unlock()

	if stopper == nil {
		return
	}

	if cancel != nil {
		cancel()
	}
	stopper.Client().Stop()
	_ = stopper.Client().Stopped().Wait()

	c.mu.Lock()
	c.state = stateStopped
	c.mu.Unlock()
}

// Status returns a snapshot of the consumer's current runtime state.
func (c *QueueConsumer[T]) Status() WorkerStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.state
	if state == "" {
		state = stateIdle
	}

	concurrency := c.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	return WorkerStatus{
		Name:        c.Name,
		Kind:        workerKindQueue,
		State:       state,
		LastRunTime: c.lastItemTime,
		LastRunDur:  c.lastItemDur,
		RunCount:    c.itemCount,
		ErrorCount:  c.errorCount,
		InFlight:    c.inFlight,
		Concurrency: concurrency,
	}
}

func (c *QueueConsumer[T]) loop(ctx context.Context, stopper concurrency.Stopper) {
	defer stopper.Flow().ReportStopped()

	concurrency := max(1, c.Concurrency)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-stopper.Flow().StopRequested():
			wg.Wait()
			return
		case item, ok := <-c.Source:
			if !ok {
				wg.Wait()
				return
			}
			sem <- struct{}{}
			wg.Add(1)
			go c.handleOne(ctx, item, sem, &wg)
		}
	}
}

func (c *QueueConsumer[T]) handleOne(ctx context.Context, item T, sem chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() { <-sem }()

	labels := prometheus.Labels{"name": c.Name, "kind": workerKindQueue}

	cur := atomic.AddInt64(&c.inFlight, 1)
	runningGauge.With(labels).Set(float64(cur))

	start := time.Now()
	err := c.Handle(ctx, item)
	dur := time.Since(start)

	cur = atomic.AddInt64(&c.inFlight, -1)
	runningGauge.With(labels).Set(float64(cur))

	runsTotal.With(labels).Inc()
	runDuration.With(labels).Observe(dur.Seconds())
	if err != nil {
		errorsTotal.With(labels).Inc()
	}

	c.mu.Lock()
	c.lastItemTime = start
	c.lastItemDur = dur
	c.itemCount++
	if err != nil {
		c.errorCount++
	}
	c.mu.Unlock()
}
