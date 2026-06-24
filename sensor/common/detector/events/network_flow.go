package events

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// NetworkFlowEvent holds the enriched state for a network flow event.
type NetworkFlowEvent struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Flow       *augmentedobjs.NetworkFlowDetails
	Netpols    *augmentedobjs.NetworkPoliciesApplied
}

func (e *NetworkFlowEvent) Topic() pubsub.Topic {
	return pubsub.DetectorNetworkFlowTopic
}

func (e *NetworkFlowEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorNetworkFlowLane
}
