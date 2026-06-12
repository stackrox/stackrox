package events

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// DeploymentEvent holds a deployment for processing through the detection pipeline.
type DeploymentEvent struct {
	Ctx        context.Context
	Deployment *storage.Deployment
	Action     central.ResourceAction
}

func (e *DeploymentEvent) Topic() pubsub.Topic {
	return pubsub.DetectorDeploymentTopic
}

func (e *DeploymentEvent) Lane() pubsub.LaneID {
	return pubsub.DetectorDeploymentLane
}
