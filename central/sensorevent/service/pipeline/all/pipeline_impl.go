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
func (s *pipelineImpl) Run(event *v1.SensorEvent, injector pipeline.EnforcementInjector) error {
	var p pipeline.Pipeline
	switch x := event.Resource.(type) {
	case *v1.SensorEvent_Deployment:
		p = s.deploymentPipeline
	case *v1.SensorEvent_NetworkPolicy:
		p = s.networkPolicyPipeline
	case *v1.SensorEvent_Namespace:
		p = s.namespacePipeline
	case *v1.SensorEvent_ProcessIndicator:
		p = s.processIndicatorPipeline
	case *v1.SensorEvent_Secret:
		p = s.secretPipeline
	case nil:
		return fmt.Errorf("Resource field is empty")
	default:
		return fmt.Errorf("No resource with type %T", x)
	}
	return p.Run(event, injector)
}
