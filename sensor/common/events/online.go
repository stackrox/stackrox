package events

import "github.com/stackrox/rox/sensor/common/pubsub"

// SensorOnlineEvent is published when the Sensor-Central gRPC connection is reachable and ready.
type SensorOnlineEvent struct{}

func (e *SensorOnlineEvent) Topic() pubsub.Topic { return pubsub.SensorOnlineTopic }
func (e *SensorOnlineEvent) Lane() pubsub.LaneID { return pubsub.SensorOnlineLane }
