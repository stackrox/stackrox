package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

// TODO: Merge this with resources.helper

func wrapOutputMessage(sensorMessages []*central.SensorEvent, detectionDeployment []message.CompatibilityDetectionMessage, reprocessDeploymentsIds []string) *message.ResourceEvent {
	return &message.ResourceEvent{
		ForwardMessages:                  sensorMessages,
		CompatibilityDetectionDeployment: detectionDeployment,
		ReprocessDeployments:             reprocessDeploymentsIds,
	}
}
