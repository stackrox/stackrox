package events

import (
	"context"
)

// LifecycleEvent is a shared base for lifecycle control signals (SoftRestart, ResourceSyncFinished).
// Each concrete event embeds this and defines only its Topic() and Lane().
type LifecycleEvent struct {
	Text     string
	Validity context.Context
}

// IsExpired reports whether the event's validity context has been cancelled.
func (e *LifecycleEvent) IsExpired() bool {
	if e.Validity == nil {
		return false
	}
	select {
	case <-e.Validity.Done():
		return true
	default:
		return false
	}
}
