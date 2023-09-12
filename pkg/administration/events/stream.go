package events

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/retry"
)

// Stream is an interface for the administration events stream.
type Stream interface {
	Consume() <-chan *AdministrationEvent
	Produce(event *AdministrationEvent) error
}

// newStream creates a new event stream.
func newStream() Stream {
	return &streamImpl{
		eventChan: make(chan *AdministrationEvent, 100),
	}
}

type streamImpl struct {
	eventChan chan *AdministrationEvent
}

// Consume returns the channel to retrieve administration events.
func (s *streamImpl) Consume() <-chan *AdministrationEvent {
	if s == nil {
		return nil
	}
	return s.eventChan
}

// Produce adds an event to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(event))`.
func (s *streamImpl) Produce(event *AdministrationEvent) error {
	if s == nil {
		return nil
	}
	select {
	case s.eventChan <- event:
		return nil
	default:
		return retry.MakeRetryable(errors.New("failed to add administration event to the stream"))
	}
}
