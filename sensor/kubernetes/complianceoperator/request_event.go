package complianceoperator

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ComplianceOperatorRequestEvent carries a ComplianceRequest handed off from Central to the
// handler's own processing loop.
type ComplianceOperatorRequestEvent struct {
	Request *central.ComplianceRequest
}

func (e *ComplianceOperatorRequestEvent) Topic() pubsub.Topic {
	return pubsub.ComplianceOperatorRequestTopic
}

func (e *ComplianceOperatorRequestEvent) Lane() pubsub.LaneID {
	return pubsub.ComplianceOperatorRequestLane
}
