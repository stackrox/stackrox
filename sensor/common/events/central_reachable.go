package events

import "github.com/stackrox/rox/sensor/common/pubsub"

// CentralReachableEvent is published when the Sensor-Central gRPC connection is reachable and ready.
type CentralReachableEvent struct{}

func (e *CentralReachableEvent) Topic() pubsub.Topic { return pubsub.CentralReachableTopic }
func (e *CentralReachableEvent) Lane() pubsub.LaneID { return pubsub.CentralReachableLane }
