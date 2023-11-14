package queue

import (
	"container/list"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Queue provides a thread-safe queue for type T.
// The queue allows to push, pull, and blocking pull.
// Additionally, it exposes safety guards such as a max size as well as metrics to track the queue growth and size.
type Queue[T comparable] struct {
	maxSize        int
	counterMetric  *prometheus.CounterVec
	queue          *list.List
	notEmptySignal concurrency.Signal
	mutex          sync.Mutex
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

// WithMaxSize provides a limit to the size of the queue. By default, no size limit is set so the queue is
// unbounded.
func WithMaxSize[T comparable](size int) OptionFunc[T] {
	return func(queue *Queue[T]) {
		queue.maxSize = size
	}
}

// NewQueue creates a new queue. Optionally, a metric can be included.
func NewQueue[T comparable](opts ...OptionFunc[T]) *Queue[T] {
	queue := &Queue[T]{
		notEmptySignal: concurrency.NewSignal(),
		queue:          list.New(),
	}

	for _, opt := range opts {
		opt(queue)
	}

	return queue
}

// Pull will pull an item from the queue. If the queue is empty, the default value of T will be returned.
// Note that his does not wait for items to be available in the queue, use PullBlocking instead.
func (q *Queue[T]) Pull() T {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() == 0 {
		var nilT T
		return nilT
	}

	item := q.queue.Remove(q.queue.Front()).(T)

	if q.counterMetric != nil {
		q.counterMetric.With(prometheus.Labels{"Operation": metrics.Remove.String()}).Inc()
	}

	if q.queue.Len() == 0 {
		q.notEmptySignal.Reset()
	}

	return item
}

// PullBlocking will pull an item from the queue, potentially waiting until one is available.
// In case the waitable signals done, the default value of T will be returned.
func (q *Queue[T]) PullBlocking(waitable concurrency.Waitable) T {
	var item T
	// In case multiple go routines are pull blocking, we have to ensure that the result of pull
	// is non-zero, hence the additional for loop here.
	for item == *new(T) {
		select {
		case <-waitable.Done():
			return item
		case <-q.notEmptySignal.Done():
			item = q.Pull()
		}
	}
	return item
}

// Push adds an item to the queue.
// Note that in case the queue is full, no error will be returned but rather only a log emitted.
func (q *Queue[T]) Push(item T) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.maxSize != 0 && q.queue.Len() >= q.maxSize {
		log.Warnf("Queue size limit reached (%d). New items added to the queue will be dropped", q.maxSize)
		return
	}

	defer q.notEmptySignal.Signal()
	if q.counterMetric != nil {
		q.counterMetric.With(prometheus.Labels{"Operation": metrics.Add.String()}).Inc()
	}
	q.queue.PushBack(item)
}
