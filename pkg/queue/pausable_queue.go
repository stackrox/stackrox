package queue

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type PausableQueue[T comparable] interface {
	Push(T)
	Pull() T
	PullBlocking() T
	Pause()
	Resume()
	Stop()
	AddAggregator(AggregateFunc[T])
	SetSize(int)
}

type AggregateFunc[T comparable] func(x, y T) (T, bool)

type pausableQueueImpl[T comparable] struct {
	mu            sync.Mutex
	internalQueue *Queue[T]
	isRunning     concurrency.Signal
	stop          concurrency.Signal
	aggregators   []AggregateFunc[T]
}

type PausableQueueOption[T comparable] func(PausableQueue[T])

func WithAggregator[T comparable](aggregator AggregateFunc[T]) PausableQueueOption[T] {
	return func(q PausableQueue[T]) {
		q.AddAggregator(aggregator)
	}
}

func WithPausableMaxSize[T comparable](size int) PausableQueueOption[T] {
	return func(q PausableQueue[T]) {
		q.SetSize(size)
	}
}

func NewPausableQueue[T comparable](opts ...PausableQueueOption[T]) PausableQueue[T] {
	q := &pausableQueueImpl[T]{
		internalQueue: NewQueue[T](),
		isRunning:     concurrency.NewSignal(),
		stop:          concurrency.NewSignal(),
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

func (q *pausableQueueImpl[T]) SetSize(size int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.internalQueue.maxSize = size
}

func (q *pausableQueueImpl[T]) AddAggregator(aggregator AggregateFunc[T]) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.aggregators = append(q.aggregators, aggregator)
}

func (q *pausableQueueImpl[T]) Push(item T) {
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

func (q *pausableQueueImpl[T]) Pull() T {
	var ret T
	if !q.isRunning.IsDone() {
		return ret
	}

	return q.internalQueue.Pull()
}

func (q *pausableQueueImpl[T]) PullBlocking() T {
	var ret T
	select {
	case <-q.stop.Done():
		return ret
	case <-q.isRunning.Done():
		return q.internalQueue.PullBlocking(&q.stop)
	}
}

func (q *pausableQueueImpl[T]) Pause() {
	q.isRunning.Reset()
}

func (q *pausableQueueImpl[T]) Resume() {
	q.isRunning.Signal()
}

func (q *pausableQueueImpl[T]) Stop() {
	q.stop.Signal()
}
