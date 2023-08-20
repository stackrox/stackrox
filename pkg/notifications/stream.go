package notifications

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
)

// Stream is an interface for the notifications stream.
type Stream interface {
	Consume() <-chan *storage.Notification
	Produce(event *storage.Notification) error
}

// NewStream creates a new notification stream.
func NewStream() Stream {
	return &streamImpl{
		notificationChan: make(chan *storage.Notification, 100),
	}
}

type streamImpl struct {
	notificationChan chan *storage.Notification
}

// Consume returns the channel to retrieve notifications.
func (s *streamImpl) Consume() <-chan *storage.Notification {
	if s == nil {
		return nil
	}
	return s.notificationChan
}

// Produce adds a notification to the stream.
//
// Should be retried with `retry.WithRetry(s.Produce(notification))`.
func (s *streamImpl) Produce(notification *storage.Notification) error {
	if s == nil {
		return nil
	}
	select {
	case s.notificationChan <- notification:
		return nil
	default:
		return retry.MakeRetryable(errors.New("failed to add notification to the stream"))
	}
}
