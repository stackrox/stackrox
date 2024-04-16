package internalmessage

import "context"

// SensorInternalMessage is the interface of messages used by publishers and subscribers to exchange messages.
type SensorInternalMessage interface {
	Kind() string
	IsExpired() bool
}

// SensorInternalTextMessage is the implementation data structure for text based internal messages.
type SensorInternalTextMessage struct {
	kind     string
	validity context.Context

	Text string
}

func (im *SensorInternalTextMessage) Kind() string {
	return im.kind
}

// IsExpired is a helper function that checks if the context already expired without blocking.
// If the context isn't set this function will always return false.
func (im *SensorInternalTextMessage) IsExpired() bool {

	if im.validity == nil {
		return false
	}

	select {
	case <-im.validity.Done():
		return true
	default:
		return false
	}
}
