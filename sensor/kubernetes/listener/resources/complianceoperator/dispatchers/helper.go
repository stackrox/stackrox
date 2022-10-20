package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
)

// TODO: Merge this with resources.helper

func wrapOutputMessage(sensorMessages []*central.SensorEvent, detectionDeployment []output.CompatibilityDetectionMessage, reprocessDeploymentsIds []string) *output.OutputMessage {
	return &output.OutputMessage{
		ForwardMessages:                  sensorMessages,
		CompatibilityDetectionDeployment: detectionDeployment,
		ReprocessDeployments:             reprocessDeploymentsIds,
	}
}
