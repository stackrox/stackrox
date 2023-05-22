package queue

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// AcceptsKeyValue is an interface that accepts a key and it's proto value for processing.
type AcceptsKeyValue interface {
	Push(key []byte, value proto.Message)
}

// WaitableQueue is a thread safe queue with an extra provided function that allows you to wait for a value to pop.
//
//go:generate mockgen-wrapper
type WaitableQueue interface {
	Push(key []byte, value proto.Message)
	PushSignal(signal *concurrency.Signal)

	Pop() ([]byte, proto.Message, *concurrency.Signal)

	Length() int

	NotEmpty() concurrency.Waitable
}

// NewWaitableQueue return a new instance of a WaitableQueue.
func NewWaitableQueue() WaitableQueue {
	return &waitableQueueImpl{
		base:        newInternalQueue(),
		notEmptySig: concurrency.NewSignal(),
		dedupeMap:   make(map[string]*queuedItem),
	}
}

type waitableQueueImpl struct {
	lock sync.Mutex

	dedupeMap   map[string]*queuedItem
	base        internalQueue
	notEmptySig concurrency.Signal
}

func (q *waitableQueueImpl) NotEmpty() concurrency.Waitable {
	return q.notEmptySig.WaitC()
}

func (q *waitableQueueImpl) Push(key []byte, value proto.Message) {
	q.push(&queuedItem{
		key:   key,
		value: value,
	})
}

func (q *waitableQueueImpl) PushSignal(signal *concurrency.Signal) {
	q.push(&queuedItem{
		signal: signal,
	})
}

func (q *waitableQueueImpl) push(qi *queuedItem) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.notEmptySig.Signal()

	if len(qi.key) != 0 {
		if oldQi, ok := q.dedupeMap[string(qi.key)]; ok {
			*oldQi = *qi
			return
		}
		q.dedupeMap[string(qi.key)] = qi
	}
	q.base.push(qi)
}

func (q *waitableQueueImpl) Pop() ([]byte, proto.Message, *concurrency.Signal) {
	q.lock.Lock()
	defer q.lock.Unlock()

	qiInter := q.base.pop()
	if qiInter == nil {
		return nil, nil, nil
	}

	if q.base.length() == 0 {
		q.notEmptySig.Reset()
	}

	qi := qiInter.(*queuedItem)
	if len(qi.key) != 0 {
		delete(q.dedupeMap, string(qi.key))
	}

	return qi.key, qi.value, qi.signal
}

func (q *waitableQueueImpl) Length() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.base.length()
}

// Helper class that holds a value with a signal.
type queuedItem struct {
	key    []byte
	value  proto.Message
	signal *concurrency.Signal
}
