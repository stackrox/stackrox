package stream

import (
	"testing"

	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/concurrency"
)

// newStream creates a new event stream.
func newStream() *streamImpl {
	return &streamImpl{
		queue: newQueue(),
	}
}

// GetStreamForTesting creates a new stream for testing purposes.
func GetStreamForTesting(_ *testing.T) *streamImpl {
	return newStream()
}

type streamImpl struct {
	queue *administrationEventsQueue
}

// Consume returns an event.
// Note that this is blocking and waits for events to be emitted before returning.
func (s *streamImpl) Consume(waitable concurrency.Waitable) *events.AdministrationEvent {
	return s.queue.pullBlocking(waitable)
}

// Produce adds an event to the stream.
func (s *streamImpl) Produce(event *events.AdministrationEvent) {
	s.queue.push(event)
}
