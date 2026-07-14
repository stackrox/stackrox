package events

import "github.com/stackrox/rox/sensor/common/pubsub"

// SyncFinishedEvent is published when Sensor completes its initial sync with Central.
type SyncFinishedEvent struct{}

func (e *SyncFinishedEvent) Topic() pubsub.Topic { return pubsub.SyncFinishedTopic }
func (e *SyncFinishedEvent) Lane() pubsub.LaneID { return pubsub.SyncFinishedLane }
