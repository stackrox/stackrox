package centralevents

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
)

// Stream is an interface for the Central events stream.
type Stream interface {
	Consume() <-chan *storage.CentralEvent
	Produce(event *storage.CentralEvent) error
}

func newStream() Stream {
	return &streamImpl{
		eventChan: make(chan *storage.CentralEvent, 100),
	}
}

type streamImpl struct {
	eventChan chan *storage.CentralEvent
}

// Consume returns the channel to retrieve Central events.
func (s *streamImpl) Consume() <-chan *storage.CentralEvent {
	return s.eventChan
}

// Produce adds an event to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(event))`.
func (s *streamImpl) Produce(event *storage.CentralEvent) error {
	select {
	case s.eventChan <- event:
		return nil
	default:
		return retry.MakeRetryable(errors.New("failed to add Central event to the stream"))
	}
}
