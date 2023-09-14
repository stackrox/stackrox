package streams

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

// Consume returns the channel to retrieve administration events.
func (s *streamImpl) Consume(waitable concurrency.Waitable) *events.AdministrationEvent {
	return s.queue.pullBlocking(waitable)
}

// Produce adds an event to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(event))`.
func (s *streamImpl) Produce(event *events.AdministrationEvent) {
	s.queue.push(event)
}
