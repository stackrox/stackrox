package events

import (
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ResourceSyncFinishedEvent is published when the initial Kubernetes resource sync completes.
type ResourceSyncFinishedEvent struct {
	LifecycleEvent
}

func (e *ResourceSyncFinishedEvent) Topic() pubsub.Topic { return pubsub.ResourceSyncFinishedTopic }
func (e *ResourceSyncFinishedEvent) Lane() pubsub.LaneID { return pubsub.ResourceSyncFinishedLane }
