package dedupingqueue

import (
	"container/list"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

// Item defines the interface that the queue items need to implement
type Item[K comparable] interface {
	GetDedupeKey() K
}

// Mergeable is an optional interface that queue items can implement to merge
// state from an existing item when a duplicate key is pushed. If the new item
// implements Mergeable, MergeFrom is called with the old item before the old
// item is removed from the queue.
type Mergeable[K comparable] interface {
	MergeFrom(old Item[K])
}

// OptionFunc provides options for the queue.
type OptionFunc[K comparable] func(*DedupingQueue[K])

// WithSizeMetrics provides a gauge to track the size of the queue.
func WithSizeMetrics[K comparable](metric prometheus.Gauge) OptionFunc[K] {
	return func(queue *DedupingQueue[K]) {
		queue.sizeMetric = metric
	}
}

// WithOperationMetricsFunc provides a function to increment a counter vector to track the queue's operations.
func WithOperationMetricsFunc[K comparable](metricFn func(ops.Op, string)) OptionFunc[K] {
	return func(queue *DedupingQueue[K]) {
		queue.operationMetric = metricFn
	}
}

// WithQueueName provides a name for the queue. This is useful for logging if there are multiple queue in use.
func WithQueueName[K comparable](name string) OptionFunc[K] {
	return func(queue *DedupingQueue[K]) {
		queue.name = name
	}
}

// WithMaxSize caps the number of items the queue retains. When a push would exceed the
// cap, the oldest item is evicted to make room, keeping the most recent items. A value
// of 0 (the default) leaves the queue unbounded.
func WithMaxSize[K comparable](maxSize int) OptionFunc[K] {
	return func(queue *DedupingQueue[K]) {
		queue.maxSize = maxSize
	}
}

// DedupingQueue a queue with unique values.
type DedupingQueue[K comparable] struct {
	lock            sync.Mutex
	notEmpty        concurrency.Signal
	queue           *list.List
	indexer         map[K]*list.Element
	sizeMetric      prometheus.Gauge
	operationMetric func(ops.Op, string)
	name            string
	// maxSize caps the number of retained items; 0 means unbounded. See WithMaxSize.
	maxSize int
}

// NewDedupingQueue creates a new DedupingQueue
func NewDedupingQueue[K comparable](opts ...OptionFunc[K]) *DedupingQueue[K] {
	ret := &DedupingQueue[K]{
		notEmpty: concurrency.NewSignal(),
		queue:    list.New(),
		indexer:  make(map[K]*list.Element),
	}
	for _, o := range opts {
		o(ret)
	}
	return ret
}

// PullBlocking blocking function that pull an item from the queue.
// If the stop signal is triggered, nil will be returned.
func (q *DedupingQueue[K]) PullBlocking(stop concurrency.Waitable) Item[K] {
	var ret Item[K]
	for ret == nil {
		select {
		case <-stop.Done():
			return nil
		case <-q.notEmpty.Done():
			ret = q.pull()
		}
	}
	return ret
}

func (q *DedupingQueue[K]) pull() Item[K] {
	q.lock.Lock()
	defer q.lock.Unlock()
	defer func() {
		if q.sizeMetric != nil {
			q.sizeMetric.Set(float64(q.queue.Len()))
		}
	}()

	if q.queue.Len() == 0 {
		return nil
	}
	ret := q.queue.Remove(q.queue.Front()).(Item[K])
	if q.operationMetric != nil {
		q.operationMetric(ops.Remove, q.name)
	}
	key := ret.GetDedupeKey()
	if key != *new(K) {
		delete(q.indexer, key)
	}
	if q.queue.Len() == 0 {
		q.notEmpty.Reset()
	}
	return ret
}

// Push adds an item to the queue if the item is not in the queue already
func (q *DedupingQueue[K]) Push(item Item[K]) {
	q.lock.Lock()
	defer q.lock.Unlock()
	defer q.notEmpty.Signal()
	defer func() {
		if q.sizeMetric != nil {
			q.sizeMetric.Set(float64(q.queue.Len()))
		}
	}()
	key := item.GetDedupeKey()
	if key == *new(K) {
		if q.operationMetric != nil {
			q.operationMetric(ops.Add, q.name)
		}
		q.queue.PushBack(item)
	} else if oldItem, ok := q.indexer[key]; ok {
		if q.operationMetric != nil {
			q.operationMetric(ops.Dedupe, q.name)
		}
		if m, ok := item.(Mergeable[K]); ok {
			m.MergeFrom(oldItem.Value.(Item[K]))
		}
		q.indexer[key] = q.queue.InsertBefore(item, oldItem)
		q.queue.Remove(oldItem)
	} else {
		if q.operationMetric != nil {
			q.operationMetric(ops.Add, q.name)
		}
		q.indexer[key] = q.queue.PushBack(item)
	}

	// Enforce the cap (if any) by evicting the oldest item. Only a unique add grows the
	// queue, so at most one eviction is ever needed per push, and the item just pushed
	// (at the back) is never the one evicted.
	if q.maxSize > 0 && q.queue.Len() > q.maxSize {
		q.evictOldestNoLock()
	}
}

// evictOldestNoLock removes the item at the front of the queue to enforce maxSize.
// The caller must hold q.lock. The eviction is reported via the Dropped operation
// metric so that overflow is observable.
func (q *DedupingQueue[K]) evictOldestNoLock() {
	front := q.queue.Front()
	if front == nil {
		return
	}
	evicted := q.queue.Remove(front).(Item[K])
	if q.operationMetric != nil {
		q.operationMetric(ops.Dropped, q.name)
	}
	if key := evicted.GetDedupeKey(); key != *new(K) {
		delete(q.indexer, key)
	}
}
