package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
)

func wrapOutputMessage(sensorMessages []*central.SensorEvent, detectionDeployment []output.CompatibilityDetectionMessage, reprocessDeploymentsIds []string) *output.Message {
	return &output.Message{
		ForwardMessages:                  sensorMessages,
		CompatibilityDetectionDeployment: detectionDeployment,
		ReprocessDeployments:             reprocessDeploymentsIds,
	}
}

func mergeOutputMessages(dest, src *output.Message) *output.Message {
	if dest == nil {
		dest = &output.Message{}
	}

	if src != nil {
		dest.ReprocessDeployments = append(dest.ReprocessDeployments, src.ReprocessDeployments...)
		dest.ForwardMessages = append(dest.ForwardMessages, src.ForwardMessages...)
		dest.CompatibilityDetectionDeployment = append(dest.CompatibilityDetectionDeployment, src.CompatibilityDetectionDeployment...)
	}
	return dest
}
