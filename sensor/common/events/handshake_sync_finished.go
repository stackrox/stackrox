package events

import "github.com/stackrox/rox/sensor/common/pubsub"

// HandshakeSyncFinishedEvent is published when Sensor completes its initial handshake sync with Central.
type HandshakeSyncFinishedEvent struct{}

func (e *HandshakeSyncFinishedEvent) Topic() pubsub.Topic { return pubsub.HandshakeSyncFinishedTopic }
func (e *HandshakeSyncFinishedEvent) Lane() pubsub.LaneID { return pubsub.HandshakeSyncFinishedLane }
