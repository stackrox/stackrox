package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/queue"
)

// SimpleQueue defines the pkg/queue that holds the items.
type SimpleQueue[T comparable] interface {
	Push(T)
	PullBlocking(concurrency.Waitable) T
	Len() int
}

// Queue wraps a SimpleQueue to make it pullable with a channel.
type Queue[T comparable] struct {
	queue     SimpleQueue[T]
	outputC   chan T
	stopper   concurrency.Stopper
	isRunning concurrency.Signal
}

// NewQueue creates a new Queue.
func NewQueue[T comparable](stopper concurrency.Stopper, name string, size int, counter *prometheus.CounterVec, dropped prometheus.Counter) *Queue[T] {
	var opts []queue.OptionFunc[T]
	if size > 0 {
		opts = append(opts, queue.WithMaxSize[T](size))
	}
	if counter != nil {
		opts = append(opts, queue.WithCounterVec[T](counter))
	}
	if dropped != nil {
		opts = append(opts, queue.WithDroppedMetric[T](dropped))
	}
	if name != "" {
		opts = append(opts, queue.WithQueueName[T](name))
	}
	return &Queue[T]{
		queue:     queue.NewQueue[T](opts...),
		outputC:   make(chan T),
		stopper:   stopper,
		isRunning: concurrency.NewSignal(),
	}
}

// Start the queue.
func (q *Queue[T]) Start() {
	// If v3 is not enabled we need to trigger isRunning here.
	if !features.SensorCapturesIntermediateEvents.Enabled() {
		q.isRunning.Signal()
	}
	go q.run()
}

// Push an item to the queue.
func (q *Queue[T]) Push(item T) {
	q.queue.Push(item)
}

func (q *Queue[T]) run() {
	defer close(q.outputC)
	for {
		select {
		case <-q.stopper.Flow().StopRequested():
			return
		case <-q.isRunning.Done():
			select {
			case <-q.stopper.Flow().StopRequested():
				return
			case q.outputC <- q.queue.PullBlocking(q.stopper.LowLevel().GetStopRequestSignal()):
			}
		}
	}
}

// Pause the queue.
func (q *Queue[T]) Pause() {
	q.isRunning.Reset()
}

// Resume the queue.
func (q *Queue[T]) Resume() {
	q.isRunning.Signal()
}

// Pull returns the channel where run writes the front of the queue.
func (q *Queue[T]) Pull() <-chan T {
	return q.outputC
}
