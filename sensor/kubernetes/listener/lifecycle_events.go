package listener

import (
	"context"

	"github.com/stackrox/rox/sensor/common/pubsub"
)

// SoftRestartEvent is published when a CRD watcher triggers a connection soft restart.
type SoftRestartEvent struct {
	Text     string
	Validity context.Context
}

func (e *SoftRestartEvent) Topic() pubsub.Topic { return pubsub.SoftRestartTopic }
func (e *SoftRestartEvent) Lane() pubsub.LaneID { return pubsub.SoftRestartLane }

// ResourceSyncFinishedEvent is published when the initial Kubernetes resource sync completes.
type ResourceSyncFinishedEvent struct {
	Text     string
	Validity context.Context
}

func (e *ResourceSyncFinishedEvent) Topic() pubsub.Topic { return pubsub.ResourceSyncFinishedTopic }
func (e *ResourceSyncFinishedEvent) Lane() pubsub.LaneID { return pubsub.ResourceSyncFinishedLane }
