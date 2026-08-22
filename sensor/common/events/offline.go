package events

import "github.com/stackrox/rox/sensor/common/pubsub"

// SensorOfflineEvent is published when the Sensor-Central connection is broken and Sensor should operate in offline mode.
type SensorOfflineEvent struct{}

func (e *SensorOfflineEvent) Topic() pubsub.Topic { return pubsub.SensorOfflineTopic }
func (e *SensorOfflineEvent) Lane() pubsub.LaneID { return pubsub.SensorOfflineLane }
