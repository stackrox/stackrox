package events

import (
	"context"

	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ResourceSyncFinishedEvent is published when the initial Kubernetes resource sync completes.
type ResourceSyncFinishedEvent struct {
	Text     string
	Validity context.Context
}

func (e *ResourceSyncFinishedEvent) Topic() pubsub.Topic { return pubsub.ResourceSyncFinishedTopic }
func (e *ResourceSyncFinishedEvent) Lane() pubsub.LaneID { return pubsub.ResourceSyncFinishedLane }

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
