package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// AuditLogManagerEvent carries an audit log message from a compliance node, routed to the
// AuditLogCollectionManager to update its per-node file state tracking.
type AuditLogManagerEvent struct {
	Node        string
	AuditEvents *sensor.AuditEvents
}

func (e *AuditLogManagerEvent) Topic() pubsub.Topic {
	return pubsub.AuditLogManagerTopic
}

func (e *AuditLogManagerEvent) Lane() pubsub.LaneID {
	return pubsub.AuditLogManagerLane
}
