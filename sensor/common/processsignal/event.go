package processsignal

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

type UnenrichedProcessIndicatorEvent struct {
	Indicator *storage.ProcessIndicator
	Context   context.Context
}

func NewUnenrichedProcessIndicatorEvent(ctx context.Context, indicator *storage.ProcessIndicator) *UnenrichedProcessIndicatorEvent {
	return &UnenrichedProcessIndicatorEvent{
		Indicator: indicator,
		Context:   ctx,
	}
}

func (e *UnenrichedProcessIndicatorEvent) Topic() pubsub.Topic {
	return pubsub.UnenrichedProcessIndicatorTopic
}

func (e *UnenrichedProcessIndicatorEvent) Lane() pubsub.LaneID {
	return pubsub.UnenrichedProcessIndicatorLane
}

type EnrichedProcessIndicatorEvent struct {
	Indicator *storage.ProcessIndicator
	Context   context.Context
}

func NewEnrichedProcessIndicatorEvent(ctx context.Context, indicator *storage.ProcessIndicator) *EnrichedProcessIndicatorEvent {
	return &EnrichedProcessIndicatorEvent{
		Indicator: indicator,
		Context:   ctx,
	}
}

func (e *EnrichedProcessIndicatorEvent) Topic() pubsub.Topic {
	return pubsub.EnrichedProcessIndicatorTopic
}

func (e *EnrichedProcessIndicatorEvent) Lane() pubsub.LaneID {
	return pubsub.EnrichedProcessIndicatorLane
}
