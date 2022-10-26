package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

func wrapOutputMessage(sensorMessages []*central.SensorEvent, detectionDeployment []message.CompatibilityDetectionMessage, reprocessDeploymentsIds []string) *message.ResourceEvent {
	return &message.ResourceEvent{
		ForwardMessages:                  sensorMessages,
		CompatibilityDetectionDeployment: detectionDeployment,
		ReprocessDeployments:             reprocessDeploymentsIds,
	}
}

func deploymentIdsMessage(ids []message.DeploymentRef) *message.ResourceEvent {
	return &message.ResourceEvent{
		DeploymentRefs: ids,
	}
}

func mergeOutputMessages(dest, src *message.ResourceEvent) *message.ResourceEvent {
	if dest == nil {
		dest = &message.ResourceEvent{}
	}

	if src != nil {
		dest.ReprocessDeployments = append(dest.ReprocessDeployments, src.ReprocessDeployments...)
		dest.ForwardMessages = append(dest.ForwardMessages, src.ForwardMessages...)
		dest.CompatibilityDetectionDeployment = append(dest.CompatibilityDetectionDeployment, src.CompatibilityDetectionDeployment...)
		dest.DeploymentRefs = append(dest.DeploymentRefs, src.DeploymentRefs...)
	}
	return dest
}
