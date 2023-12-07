package queue

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/queue"
)

// Queue wraps a PausableQueue to make it pullable with a channel.
type Queue[T comparable] struct {
	queue   queue.PausableQueue[T]
	outputC chan T
	stopper concurrency.Stopper
}

// NewQueue creates a new Queue.
func NewQueue[T comparable](stopper concurrency.Stopper) *Queue[T] {
	return &Queue[T]{
		queue:   queue.NewPausableQueue[T](),
		outputC: make(chan T),
		stopper: stopper,
	}
}

// Start the queue.
func (q *Queue[T]) Start() {
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	q.queue.Resume()
	go q.run()
}

// Push an item to the queue
func (q *Queue[T]) Push(item T) {
	q.queue.Push(item)
}

func (q *Queue[T]) run() {
	defer close(q.outputC)
	for {
		select {
		case <-q.stopper.Flow().StopRequested():
			return
		default:
			q.outputC <- q.queue.PullBlocking()
		}
	}
}

// Pause the queue.
func (q *Queue[T]) Pause() {
	q.queue.Pause()
}

// Resume the queue.
func (q *Queue[T]) Resume() {
	q.queue.Resume()
}

// Stop the queue.
func (q *Queue[T]) Stop() {
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	q.queue.Stop()
}

// Pull returns the channel where run writes the front of the queue.
func (q *Queue[T]) Pull() <-chan T {
	return q.outputC
}
