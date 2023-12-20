package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// AggregateFunc aggregates two comparable variables.
// Returns the aggregate and a boolean indicating whether the aggregation took place or not.
type AggregateFunc[T comparable] func(x, y T) (T, bool)

// PausableQueue defines a queue that can be paused.
// Given aggregate functions (AggregateFunc), pushed items can be aggregate with items already in the queue.
type PausableQueue[T comparable] struct {
	mu            sync.Mutex
	internalQueue *Queue[T]
	isRunning     concurrency.Signal
	aggregators   []AggregateFunc[T]
}

// PausableQueueOption provides options for the queue.
type PausableQueueOption[T comparable] func(*PausableQueue[T])

// WithAggregator provides an AggregateFunc for the queue.
func WithAggregator[T comparable](aggregator AggregateFunc[T]) PausableQueueOption[T] {
	return func(q *PausableQueue[T]) {
		q.aggregators = append(q.aggregators, aggregator)
	}
}

// WithPausableMaxSize provides a maximum size for the queue. By default, no size limit is set.
func WithPausableMaxSize[T comparable](size int) PausableQueueOption[T] {
	return func(q *PausableQueue[T]) {
		q.internalQueue.maxSize = size
	}
}

// WithPausableQueueMetric provides a counter vec which tracks added and removed items from the queue.
func WithPausableQueueMetric[T comparable](metric *prometheus.CounterVec) PausableQueueOption[T] {
	return func(q *PausableQueue[T]) {
		q.internalQueue.counterMetric = metric
	}
}

// WithPausableQueueDroppedMetric provides a counter which tracks number of items dropped.
func WithPausableQueueDroppedMetric[T comparable](metric prometheus.Counter) PausableQueueOption[T] {
	return func(q *PausableQueue[T]) {
		q.internalQueue.droppedMetric = metric
	}
}

// NewPausableQueue creates a new PausableQueue.
func NewPausableQueue[T comparable](opts ...PausableQueueOption[T]) *PausableQueue[T] {
	q := &PausableQueue[T]{
		internalQueue: NewQueue[T](),
		isRunning:     concurrency.NewSignal(),
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

// Push adds an item to the queue.
// If the queue has aggregator functions (AggregateFunc) the queue will try to aggregate with the items already in the queue.
// Note that we only aggregate once.
func (q *PausableQueue[T]) Push(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.aggregators) == 0 {
		q.internalQueue.Push(item)
		return
	}

	wasAggregated := false
	for e := q.internalQueue.queue.Front(); e != nil; e = e.Next() {
		element, ok := e.Value.(T)
		if !ok {
			continue
		}
		for _, fn := range q.aggregators {
			val, aggregated := fn(element, item)
			if aggregated {
				e.Value = val
				wasAggregated = true
				break
			}
		}
		if wasAggregated {
			break
		}
	}

	if !wasAggregated {
		q.internalQueue.Push(item)
	}
}

// Pull will pull an item from the queue. If the queue is empty or paused, the default value of T will be returned.
// Note that his does not wait for items to be available in the queue, use PullBlocking instead.
func (q *PausableQueue[T]) Pull() T {
	var ret T
	if !q.isRunning.IsDone() {
		return ret
	}

	return q.internalQueue.Pull()
}

// PullBlocking will pull an item from the queue, potentially waiting until one is available.
// In case the queue is stopped, the default value of T will be returned.
func (q *PausableQueue[T]) PullBlocking(waitable concurrency.Waitable) T {
	var retNil T
	ret := q.internalQueue.PullBlocking(waitable)
	select {
	case <-waitable.Done():
		return retNil
	case <-q.isRunning.Done():
		return ret
	}
}

// Pause the queue.
func (q *PausableQueue[T]) Pause() {
	q.isRunning.Reset()
}

// Resume the queue.
func (q *PausableQueue[T]) Resume() {
	q.isRunning.Signal()
}
