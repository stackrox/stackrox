package queue

import (
	"container/list"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	loggingRateLimiter = "pkg-queue"
	defaultQueueName   = "Queue"
)

// Queue provides a thread-safe queue for type T.
// The queue allows to push, pull, and blocking pull.
// Additionally, it exposes safety guards such as a max size as well as metrics to track the queue growth and size.
type Queue[T comparable] struct {
	name           string
	maxSize        int
	counterMetric  *prometheus.CounterVec
	droppedMetric  prometheus.Counter
	queue          *list.List
	notEmptySignal concurrency.Signal
	mutex          sync.Mutex
	// afterEmptyPull runs after pullWait sees an empty queue and before it waits.
	// Tests inject a Push in that window; production leaves it nil.
	afterEmptyPull func()
}

// OptionFunc provides options for the queue.
// Note that the comparable type is currently required, once we upgrade to go1.20 we can switch this to
// any and creation will be much easier.
type OptionFunc[T comparable] func(queue *Queue[T])

// WithCounterVec provides a counter vec which tracks added and removed items from the queue.
func WithCounterVec[T comparable](vec *prometheus.CounterVec) OptionFunc[T] {
	return func(queue *Queue[T]) {
		queue.counterMetric = vec
	}
}

// WithDroppedMetric provides a counter which tracks number of items dropped.
func WithDroppedMetric[T comparable](metric prometheus.Counter) OptionFunc[T] {
	return func(q *Queue[T]) {
		q.droppedMetric = metric
	}
}

// WithMaxSize provides a limit to the size of the queue. By default, no size limit is set so the queue is
// unbounded.
func WithMaxSize[T comparable](size int) OptionFunc[T] {
	return func(queue *Queue[T]) {
		queue.maxSize = size
	}
}

// WithQueueName provides a name for the queue. This is useful for logging if there are multiple queue in use.
func WithQueueName[T comparable](name string) OptionFunc[T] {
	return func(queue *Queue[T]) {
		queue.name = name
	}
}

// NewQueue creates a new queue. Optionally, a metric can be included.
func NewQueue[T comparable](opts ...OptionFunc[T]) *Queue[T] {
	queue := &Queue[T]{
		notEmptySignal: concurrency.NewSignal(),
		queue:          list.New(),
		name:           defaultQueueName,
		afterEmptyPull: func() {}, // no-op by default, tests can override
	}

	for _, opt := range opts {
		opt(queue)
	}

	return queue
}

// Pull will pull an item from the queue. If the queue is empty, the default value of T will be returned.
// Note that his does not wait for items to be available in the queue, use PullBlocking instead.
func (q *Queue[T]) Pull() T {
	item, _ := q.pull()
	return item
}

// PullBlocking will pull an item from the queue, potentially waiting until one is available.
// In case the waitable signals done, the default value of T will be returned.
func (q *Queue[T]) PullBlocking(waitable concurrency.Waitable) T {
	item, _ := q.pullWait(waitable)
	return item
}

// Seq returns a iterator function that yields items from the queue as they become available.
// The iterator will continue until the provided waitable signals done.
//
// Note: Seq checks for cancellation before each pull. If the waitable is cancelled while items
// remain in the queue, Seq will exit immediately without consuming them. This differs from
// PullBlocking, which will pull at least one item before checking cancellation.
func (q *Queue[T]) Seq(waitable concurrency.Waitable) func(yield func(T) bool) {
	return func(yield func(T) bool) {
		for {
			// Ensure responsive cancellation even when queue has items
			select {
			case <-waitable.Done():
				return
			default:
			}

			item, ok := q.pullWait(waitable)
			if !ok {
				return
			}

			if !yield(item) {
				return
			}
		}
	}
}

// pullWait waits for an item to become available in the queue, blocking until
// either an item is retrieved or the waitable signals done.
// Returns the item and true if retrieved, or zero value and false if cancelled.
func (q *Queue[T]) pullWait(waitable concurrency.Waitable) (T, bool) {
	item, ok := q.pull()
	// Keep retrying until we actually get an item or context is cancelled.
	// This prevents lost wakeup: if we're signaled but another consumer
	// takes the item before we pull, we must continue waiting.
	for !ok {
		q.afterEmptyPull()
		// No Reset() here - innerPull now resets the signal atomically
		// with observing the empty queue, eliminating the race window.
		select {
		case <-waitable.Done():
			var nilT T
			return nilT, false
		case <-q.notEmptySignal.Done():
		}
		item, ok = q.pull()
	}
	return item, true
}

func (q *Queue[T]) innerPull() (T, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() == 0 {
		// Reset signal while observing empty queue under lock.
		// This prevents the race where a Push signals after we check
		// but before we wait - the signal and our observation are atomic.
		q.notEmptySignal.Reset()
		var nilT T
		return nilT, false
	}

	item := q.queue.Remove(q.queue.Front()).(T)
	if q.queue.Len() == 0 {
		q.notEmptySignal.Reset()
	}

	return item, true
}

func (q *Queue[T]) pull() (T, bool) {
	item, ok := q.innerPull()
	if ok && q.counterMetric != nil {
		// Using `WithLabelValues` instead of `With` to avoid extra memory allocations.
		q.counterMetric.WithLabelValues(metrics.Remove.String()).Inc()
	}
	return item, ok
}

func (q *Queue[T]) innerPush(item T) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.maxSize != 0 && q.queue.Len() >= q.maxSize {
		return false
	}

	q.queue.PushBack(item)
	// Signal while holding the lock, atomically with PushBack.
	// This prevents the race where a consumer observes empty and resets
	// the signal after we've added the item but before we signal.
	q.notEmptySignal.Signal()
	return true
}

// Push adds an item to the queue.
// Note that in case the queue is full, no error will be returned but rather only a log emitted.
func (q *Queue[T]) Push(item T) {
	if !q.innerPush(item) {
		logging.GetRateLimitedLogger().WarnL(loggingRateLimiter, "Queue (%s) size limit reached (%d). New items added to the queue will be dropped.", q.name, q.maxSize)
		if q.droppedMetric != nil {
			q.droppedMetric.Inc()
		}
		return
	}

	// Signal is now sent inside innerPush under the lock
	if q.counterMetric != nil {
		// Using `WithLabelValues` instead of `With` to avoid extra memory allocations.
		q.counterMetric.WithLabelValues(metrics.Add.String()).Inc()
	}
}

// PullWithPred removes and returns the first element for which pred returns
// true, scanning front-to-back. Returns the removed item and true, or the nil
// value of T and false if no element matched.
func (q *Queue[T]) PullWithPred(pred func(T) bool) (T, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for e := q.queue.Front(); e != nil; e = e.Next() {
		item := e.Value.(T)
		if pred(item) {
			q.queue.Remove(e)
			if q.queue.Len() == 0 {
				q.notEmptySignal.Reset()
			}
			if q.counterMetric != nil {
				q.counterMetric.WithLabelValues(metrics.Remove.String()).Inc()
			}
			return item, true
		}
	}

	var nilT T
	return nilT, false
}

// Len returns the number of elements in the queue.
func (q *Queue[T]) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Len()
}
