package events

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// AuditLogEvent holds audit log events for detection.
type AuditLogEvent struct {
	AuditEvents *sensor.AuditEvents
}

func (e *AuditLogEvent) Topic() pubsub.Topic {
	return pubsub.DetectorAuditLogTopic
}

func (e *AuditLogEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorAuditLogLane
}
