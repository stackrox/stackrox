package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/queue"
)

// PausableQueue defines a queue that can be paused.
type PausableQueue[T comparable] interface {
	Push(T)
	PullBlocking(concurrency.Waitable) T
	Pause()
	Resume()
}

// Queue wraps a PausableQueue to make it pullable with a channel.
type Queue[T comparable] struct {
	queue   PausableQueue[T]
	outputC chan T
	stopper concurrency.Stopper
}

// NewQueue creates a new Queue.
func NewQueue[T comparable](stopper concurrency.Stopper, size int, counter *prometheus.CounterVec, dropped prometheus.Counter) *Queue[T] {
	var opts []queue.PausableQueueOption[T]
	if size > 0 {
		opts = append(opts, queue.WithPausableMaxSize[T](size))
	}
	if counter != nil {
		opts = append(opts, queue.WithPausableQueueMetric[T](counter))
	}
	if dropped != nil {
		opts = append(opts, queue.WithPausableQueueDroppedMetric[T](dropped))
	}
	return &Queue[T]{
		queue:   queue.NewPausableQueue[T](opts...),
		outputC: make(chan T),
		stopper: stopper,
	}
}

// Start the queue.
func (q *Queue[T]) Start() {
	// If v3 is not enabled we need to start the queue here.
	if !features.SensorCapturesIntermediateEvents.Enabled() {
		q.queue.Resume()
	}
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
			q.outputC <- q.queue.PullBlocking(q.stopper.LowLevel().GetStopRequestSignal())
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

// Pull returns the channel where run writes the front of the queue.
func (q *Queue[T]) Pull() <-chan T {
	return q.outputC
}
