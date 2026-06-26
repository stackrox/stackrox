package events

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// IndicatorEvent holds the enriched state for a process indicator event.
type IndicatorEvent struct {
	Ctx          context.Context
	Deployment   *storage.Deployment
	Indicator    *storage.ProcessIndicator
	Netpols      *augmentedobjs.NetworkPoliciesApplied
	IsInBaseline bool
}

func (e *IndicatorEvent) Topic() pubsub.Topic {
	return pubsub.DetectorProcessIndicatorTopic
}

func (e *IndicatorEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorProcessIndicatorLane
}
