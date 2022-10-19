package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
)

func wrapOutputMessage(sensorMessages []*central.SensorEvent, action central.ResourceAction, detectionDeployment *storage.Deployment, reprocessDeploymentsIds []string) *output.OutputMessage {
	return &output.OutputMessage{
		ForwardMessages:                  sensorMessages,
		Action:                           action,
		CompatibilityDetectionDeployment: detectionDeployment,
		ReprocessDeployments:             reprocessDeploymentsIds,
	}
}
