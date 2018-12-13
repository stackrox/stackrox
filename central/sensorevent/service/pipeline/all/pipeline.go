package all

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

// NewPipeline returns a new instance of a Pipeline that handles all event types.
func NewPipeline(deploymentPipeline,
	processIndicatorPipeline,
	networkPolicyPipeline,
	namespacePipeline,
	secretPipeline,
	clusterStatusPipeline,
	providerMetadataPipeline pipeline.Pipeline) pipeline.Pipeline {
	return &pipelineImpl{
		deploymentPipeline:       deploymentPipeline,
		processIndicatorPipeline: processIndicatorPipeline,
		networkPolicyPipeline:    networkPolicyPipeline,
		namespacePipeline:        namespacePipeline,
		secretPipeline:           secretPipeline,
		nodePipeline:             clusterStatusPipeline,
		providerMetadataPipeline: providerMetadataPipeline,
	}
}
