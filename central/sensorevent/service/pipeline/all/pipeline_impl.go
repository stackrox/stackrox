package all

import (
	"fmt"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
)

type pipelineImpl struct {
	deploymentPipeline       pipeline.Pipeline
	processIndicatorPipeline pipeline.Pipeline
	networkPolicyPipeline    pipeline.Pipeline
	namespacePipeline        pipeline.Pipeline
	secretPipeline           pipeline.Pipeline
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error) {
	switch x := event.Resource.(type) {
	case *v1.SensorEvent_Deployment:
		return s.deploymentPipeline.Run(event)
	case *v1.SensorEvent_NetworkPolicy:
		return s.networkPolicyPipeline.Run(event)
	case *v1.SensorEvent_Namespace:
		return s.namespacePipeline.Run(event)
	case *v1.SensorEvent_ProcessIndicator:
		return s.processIndicatorPipeline.Run(event)
	case *v1.SensorEvent_Secret:
		return s.secretPipeline.Run(event)
	case nil:
		return nil, fmt.Errorf("Resource field is empty")
	default:
		return nil, fmt.Errorf("No resource with type %T", x)
	}
}
