package queue

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
	internalQueue *Queue[T]
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

func NewPausableQueue[T comparable](_ ...PausableQueueOption[T]) PausableQueue[T] {
	return &pausableQueueImpl[T]{}
}

func (q *pausableQueueImpl[T]) SetSize(_ int) {}

func (q *pausableQueueImpl[T]) AddAggregator(_ AggregateFunc[T]) {}

func (q *pausableQueueImpl[T]) Push(_ T) {}

func (q *pausableQueueImpl[T]) Pull() T {
	var ret T
	return ret
}

func (q *pausableQueueImpl[T]) PullBlocking() T {
	var ret T
	return ret
}

func (q *pausableQueueImpl[T]) Pause() {}

func (q *pausableQueueImpl[T]) Resume() {}

func (q *pausableQueueImpl[T]) Stop() {}
