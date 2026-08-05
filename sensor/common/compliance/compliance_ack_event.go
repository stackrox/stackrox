package compliance

import (
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ComplianceAckEvent carries a ComplianceACK/NACK message destined for a compliance gRPC node.
type ComplianceAckEvent struct {
	Msg common.MessageToComplianceWithAddress
}

func (e *ComplianceAckEvent) Topic() pubsub.Topic {
	return pubsub.ComplianceAckTopic
}

func (e *ComplianceAckEvent) Lane() pubsub.LaneID {
	return pubsub.ComplianceAckLane
}
