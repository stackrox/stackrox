package events

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// DeployAlertOutputEvent carries deploy-time alert results to the serializer.
type DeployAlertOutputEvent struct {
	Results   *central.AlertResults
	Timestamp int64
	Action    central.ResourceAction
	Context   context.Context
}

func (e *DeployAlertOutputEvent) Topic() pubsub.Topic {
	return pubsub.DetectorDeployAlertOutputTopic
}

func (e *DeployAlertOutputEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorDeployAlertOutputLane
}
