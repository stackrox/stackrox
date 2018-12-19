package all

import (
	"fmt"

	pkgMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
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
	providerMetadataPipeline pipeline.Pipeline
}

func actionToOperation(action central.ResourceAction) metrics.Op {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE:
		return metrics.Add
	case central.ResourceAction_UPDATE_RESOURCE:
		return metrics.Update
	case central.ResourceAction_REMOVE_RESOURCE:
		return metrics.Remove
	default:
		log.Fatalf("Unknown action to operation '%s'", action)
	}
	// Appease the almighty compiler
	return 0
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *pipelineImpl) Run(event *central.SensorEvent, injector pipeline.EnforcementInjector) error {

	var p pipeline.Pipeline
	var resource metrics.Resource
	switch x := event.Resource.(type) {
	case *central.SensorEvent_Deployment:
		resource = metrics.Deployment
		p = s.deploymentPipeline
	case *central.SensorEvent_NetworkPolicy:
		resource = metrics.NetworkPolicy
		p = s.networkPolicyPipeline
	case *central.SensorEvent_Namespace:
		resource = metrics.Namespace
		p = s.namespacePipeline
	case *central.SensorEvent_ProcessIndicator:
		resource = metrics.ProcessIndicator
		p = s.processIndicatorPipeline
	case *central.SensorEvent_Secret:
		resource = metrics.Secret
		p = s.secretPipeline
	case *central.SensorEvent_Node:
		resource = metrics.Node
		p = s.nodePipeline
	case *central.SensorEvent_ProviderMetadata:
		resource = metrics.ProviderMetadata
		p = s.providerMetadataPipeline
	case nil:
		return fmt.Errorf("Resource field is empty")
	default:
		return fmt.Errorf("No resource with type %T", x)
	}
	pkgMetrics.IncrementResourceProcessedCounter(actionToOperation(event.GetAction()), resource)
	return p.Run(event, injector)
}
