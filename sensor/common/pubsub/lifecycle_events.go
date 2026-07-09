package pubsub

import (
	"context"
)

// SoftRestartEvent is published when a CRD watcher triggers a connection soft restart.
type SoftRestartEvent struct {
	Text     string
	Validity context.Context
}

func (e *SoftRestartEvent) Topic() Topic   { return SoftRestartTopic }
func (e *SoftRestartEvent) Lane() LaneID   { return SoftRestartLane }
func (e *SoftRestartEvent) String() string { return e.Text }

// IsExpired reports whether the event's validity context has been cancelled.
func (e *SoftRestartEvent) IsExpired() bool {
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

// ResourceSyncFinishedEvent is published when the initial Kubernetes resource sync completes.
type ResourceSyncFinishedEvent struct {
	Text     string
	Validity context.Context
}

func (e *ResourceSyncFinishedEvent) Topic() Topic { return ResourceSyncFinishedTopic }
func (e *ResourceSyncFinishedEvent) Lane() LaneID { return ResourceSyncFinishedLane }

// IsExpired reports whether the event's validity context has been cancelled.
func (e *ResourceSyncFinishedEvent) IsExpired() bool {
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
