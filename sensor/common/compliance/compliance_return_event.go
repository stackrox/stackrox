package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ComplianceReturnEvent carries a ComplianceReturn result relayed from a compliance gRPC node.
type ComplianceReturnEvent struct {
	Return *compliance.ComplianceReturn
}

func (e *ComplianceReturnEvent) Topic() pubsub.Topic {
	return pubsub.ComplianceReturnTopic
}

func (e *ComplianceReturnEvent) Lane() pubsub.LaneID {
	return pubsub.ComplianceReturnLane
}
