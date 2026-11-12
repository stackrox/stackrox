package events

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ComplianceCentralRequestEvent carries a compliance request from Central
// to the compliance manager for processing.
type ComplianceCentralRequestEvent struct {
	Context context.Context
	Request *central.ComplianceRequest
}

func (e *ComplianceCentralRequestEvent) Topic() pubsub.Topic {
	return pubsub.ComplianceCentralRequestTopic
}
func (e *ComplianceCentralRequestEvent) Lane() pubsub.LaneID {
	return pubsub.ComplianceCentralRequestLane
}
