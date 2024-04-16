package internalmessage

import "context"

// SensorInternalMessage is the implementation data structure for text based internal messages.
type SensorInternalMessage struct {
	Kind     string
	Validity context.Context
	Text     string
}

// IsExpired is a helper function that checks if the context already expired without blocking.
// If the context isn't set this function will always return false.
func (im *SensorInternalMessage) IsExpired() bool {

	if im.Validity == nil {
		return false
	}

	select {
	case <-im.Validity.Done():
		return true
	default:
		return false
	}
}
