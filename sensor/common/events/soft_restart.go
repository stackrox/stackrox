package events

import (
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// SoftRestartEvent is published when a CRD watcher triggers a connection soft restart.
type SoftRestartEvent struct {
	LifecycleEvent
}

func (e *SoftRestartEvent) Topic() pubsub.Topic { return pubsub.SoftRestartTopic }
func (e *SoftRestartEvent) Lane() pubsub.LaneID { return pubsub.SoftRestartLane }
func (e *SoftRestartEvent) String() string      { return e.Text }
