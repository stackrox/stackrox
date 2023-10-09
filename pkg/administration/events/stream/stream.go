package stream

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/queue"
)

const (
	// Sample calculation with a sample administration event (250 chars in message + hint):
	// 1 Administration event = 160 bytes
	// 100000 *160 bytes = 16 MB
	maxQueueSize = 100000
)

var (
	administrationEventsQueueCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "administration_events_queue_size_total",
		Help:      "A counter that tracks the size of the administration events queue",
	}, []string{"Operation"})
)

func init() {
	prometheus.MustRegister(administrationEventsQueueCounter)
}

// newStream creates a new event stream.
func newStream() *streamImpl {
	return &streamImpl{
		queue: queue.NewQueue[*events.AdministrationEvent](queue.WithMaxSize[*events.AdministrationEvent](maxQueueSize),
			queue.WithCounterVec[*events.AdministrationEvent](administrationEventsQueueCounter)),
	}
}

// GetStreamForTesting creates a new stream for testing purposes.
func GetStreamForTesting(_ *testing.T) *streamImpl {
	return newStream()
}

type streamImpl struct {
	queue *queue.Queue[*events.AdministrationEvent]
}

// Consume returns an event.
// Note that this is blocking and waits for events to be emitted before returning.
func (s *streamImpl) Consume(waitable concurrency.Waitable) *events.AdministrationEvent {
	return s.queue.PullBlocking(waitable)
}

// Produce adds an event to the stream.
func (s *streamImpl) Produce(event *events.AdministrationEvent) {
	s.queue.Push(event)
}
