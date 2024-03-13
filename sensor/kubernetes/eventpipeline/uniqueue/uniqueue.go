package uniqueue

import (
	"container/list"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Item defines the interface that the queue items need to implement
type Item interface {
	GetKey() string
}

// OptionFunc provides options for the queue.
type OptionFunc func(*UniQueue)

// WithMetrics provides a gauge to track the size of the queue.
func WithMetrics(metric prometheus.Gauge) OptionFunc {
	return func(queue *UniQueue) {
		queue.metric = metric
	}
}

// UniQueue a queue with unique values.
// TODO: this is very similar to the deduping queue in central. Refactor this and central's queue to a generic abstraction.
type UniQueue struct {
	lock     sync.Mutex
	notEmpty concurrency.Signal
	queue    *list.List
	indexer  map[string]*list.Element
	metric   prometheus.Gauge
}

// NewUniQueue creates a new UniQueue
func NewUniQueue(opts ...OptionFunc) *UniQueue {
	ret := &UniQueue{
		notEmpty: concurrency.NewSignal(),
		queue:    list.New(),
		indexer:  make(map[string]*list.Element),
	}
	for _, o := range opts {
		o(ret)
	}
	return ret
}

// PullBlocking blocking function that pull an item from the queue.
// If the stop signal is triggered, nil will be returned.
func (q *UniQueue) PullBlocking(stop concurrency.Waitable) Item {
	var ret Item
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

func (q *UniQueue) pull() Item {
	q.lock.Lock()
	defer q.lock.Unlock()
	defer func() {
		if q.metric != nil {
			q.metric.Set(float64(q.queue.Len()))
		}
	}()

	if q.queue.Len() == 0 {
		return nil
	}
	ret := q.queue.Remove(q.queue.Front()).(Item)
	key := ret.GetKey()
	if key != "" {
		delete(q.indexer, key)
	}
	if q.queue.Len() == 0 {
		q.notEmpty.Reset()
	}
	return ret
}

// Push adds an item to the queue if the item is not in the queue already
func (q *UniQueue) Push(item Item) {
	q.lock.Lock()
	defer q.lock.Unlock()
	defer q.notEmpty.Signal()
	defer func() {
		if q.metric != nil {
			q.metric.Set(float64(q.queue.Len()))
		}
	}()
	key := item.GetKey()
	if key == "" {
		q.queue.PushBack(item)
		return
	}
	var msgInserted *list.Element
	if oldItem, ok := q.indexer[key]; !ok {
		msgInserted = q.queue.PushBack(item)
	} else {
		msgInserted = q.queue.InsertBefore(item, oldItem)
		q.queue.Remove(oldItem)
	}
	q.indexer[key] = msgInserted
}
