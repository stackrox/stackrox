package all

import (
	"fmt"

	pkgMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var log = logging.LoggerForModule()

type pipelineImpl struct {
	deploymentPipeline       pipeline.Pipeline
	processIndicatorPipeline pipeline.Pipeline
	networkPolicyPipeline    pipeline.Pipeline
	namespacePipeline        pipeline.Pipeline
	secretPipeline           pipeline.Pipeline
	nodePipeline             pipeline.Pipeline
}

func actionToOperation(action v1.ResourceAction) metrics.Op {
	switch action {
	case v1.ResourceAction_CREATE_RESOURCE:
		return metrics.Add
	case v1.ResourceAction_UPDATE_RESOURCE:
		return metrics.Update
	case v1.ResourceAction_REMOVE_RESOURCE:
		return metrics.Remove
	default:
		log.Fatalf("Unknown action to operation '%s'", action)
	}
	// Appease the almighty compiler
	return 0
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *pipelineImpl) Run(event *v1.SensorEvent, injector pipeline.EnforcementInjector) error {

	var p pipeline.Pipeline
	var resource metrics.Resource
	switch x := event.Resource.(type) {
	case *v1.SensorEvent_Deployment:
		resource = metrics.Deployment
		p = s.deploymentPipeline
	case *v1.SensorEvent_NetworkPolicy:
		resource = metrics.NetworkPolicy
		p = s.networkPolicyPipeline
	case *v1.SensorEvent_Namespace:
		resource = metrics.Namespace
		p = s.namespacePipeline
	case *v1.SensorEvent_ProcessIndicator:
		resource = metrics.ProcessIndicator
		p = s.processIndicatorPipeline
	case *v1.SensorEvent_Secret:
		resource = metrics.Secret
		p = s.secretPipeline
	case *v1.SensorEvent_Node:
		resource = metrics.Node
		p = s.nodePipeline
	case nil:
		return fmt.Errorf("Resource field is empty")
	default:
		return fmt.Errorf("No resource with type %T", x)
	}
	pkgMetrics.IncrementResourceProcessedCounter(actionToOperation(event.GetAction()), resource)
	return p.Run(event, injector)
}
