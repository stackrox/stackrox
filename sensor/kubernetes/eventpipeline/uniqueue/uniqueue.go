package uniqueue

import (
	"container/list"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Item defines the interface that the queue items need to implement
type Item interface {
	GetKey() string
}

// UniQueue a queue with unique values.
// TODO: this is very similar to the deduping queue in central. Refactor this and central's queue to a generic abstraction.
type UniQueue struct {
	lock     sync.Mutex
	notEmpty concurrency.Signal
	queue    *list.List
	indexer  map[string]*list.Element
}

// NewUniQueue creates a new UniQueue
func NewUniQueue() *UniQueue {
	return &UniQueue{
		notEmpty: concurrency.NewSignal(),
		queue:    list.New(),
		indexer:  make(map[string]*list.Element),
	}
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
