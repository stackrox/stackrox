package events

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
)

// Stream is an interface for the administration events stream.
type Stream interface {
	Consume() <-chan *storage.AdministrationEvent
	Produce(event *storage.AdministrationEvent) error
}

func newStream() Stream {
	return &streamImpl{
		eventChan: make(chan *storage.AdministrationEvent, 100),
	}
}

type streamImpl struct {
	eventChan chan *storage.AdministrationEvent
}

// Consume returns the channel to retrieve administration events.
func (s *streamImpl) Consume() <-chan *storage.AdministrationEvent {
	return s.eventChan
}

// Produce adds an event to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(event))`.
func (s *streamImpl) Produce(event *storage.AdministrationEvent) error {
	select {
	case s.eventChan <- event:
		return nil
	default:
		return retry.MakeRetryable(errors.New("failed to add administration event to the stream"))
	}
}
