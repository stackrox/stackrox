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
func (n *Queue[T]) Start() {
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	n.queue.Resume()
	go n.run()
}

// Push an item to the queue
func (n *Queue[T]) Push(item T) {
	n.queue.Push(item)
}

func (n *Queue[T]) run() {
	defer close(n.outputC)
	// TODO(ROX-21052): Resuming, pausing, and stopping the internal queue should be done in the QueueManager
	defer n.queue.Stop()
	for {
		select {
		case <-n.stopper.Flow().StopRequested():
			return
		default:
			n.outputC <- n.queue.PullBlocking()
		}
	}
}

// Pull returns the channel where run writes the front of the queue.
func (n *Queue[T]) Pull() <-chan T {
	return n.outputC
}
