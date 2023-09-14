package events

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency"
)

// Stream is an interface for the administration events stream.
type Stream interface {
	Consume(waitable concurrency.Waitable) *AdministrationEvent
	Produce(event *AdministrationEvent)
}

// newStream creates a new event stream.
func newStream() Stream {
	return &streamImpl{
		queue: newQueue(),
	}
}

// GetStreamForTesting creates a new stream for testing purposes.
func GetStreamForTesting(_ *testing.T) Stream {
	return newStream()
}

type streamImpl struct {
	queue *administrationEventsQueue
}

// Consume returns the channel to retrieve administration events.
func (s *streamImpl) Consume(waitable concurrency.Waitable) *AdministrationEvent {
	return s.queue.pullBlocking(waitable)
}

// Produce adds an event to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(event))`.
func (s *streamImpl) Produce(event *AdministrationEvent) {
	s.queue.push(event)
}
