package all

import (
	"context"

	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/alerts"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/auditlogstateupdate"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/clusterhealthupdate"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/clustermetrics"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/clusterstatusupdate"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/complianceoperatorprofiles"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/complianceoperatorresults"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/complianceoperatorrules"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/complianceoperatorscans"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/complianceoperatorscansettingbinding"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/deploymentevents"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/imageintegrations"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/namespaces"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/networkflowupdate"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/networkpolicies"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/podevents"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/processindicators"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reprocessing"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/rolebindings"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/roles"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/secrets"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/serviceaccounts"
	"github.com/stackrox/stackrox/pkg/features"
)

// NewFactory returns a new instance of a Factory that produces a pipeline handling all message types.
func NewFactory() pipeline.Factory {
	return &factoryImpl{}
}

type factoryImpl struct{}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *factoryImpl) PipelineForCluster(ctx context.Context, clusterID string) (pipeline.ClusterPipeline, error) {
	flowUpdateFragment, err := networkflowupdate.Singleton().GetFragment(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pipelines := []pipeline.Fragment{
		deploymentevents.GetPipeline(),
		podevents.GetPipeline(),
		processindicators.GetPipeline(),
		networkpolicies.GetPipeline(),
		namespaces.GetPipeline(),
		secrets.GetPipeline(),
		nodes.GetPipeline(),
		flowUpdateFragment,
		imageintegrations.GetPipeline(),
		clusterstatusupdate.GetPipeline(),
		clusterhealthupdate.GetPipeline(),
		clustermetrics.GetPipeline(),
		serviceaccounts.GetPipeline(),
		roles.GetPipeline(),
		rolebindings.GetPipeline(),
		reprocessing.GetPipeline(),
		alerts.GetPipeline(),
		auditlogstateupdate.GetPipeline(),
	}
	if features.ComplianceOperatorCheckResults.Enabled() {
		pipelines = append(pipelines,
			complianceoperatorresults.GetPipeline(),
			complianceoperatorprofiles.GetPipeline(),
			complianceoperatorscansettingbinding.GetPipeline(),
			complianceoperatorrules.GetPipeline(),
			complianceoperatorscans.GetPipeline(),
		)
	}

	return NewClusterPipeline(clusterID, pipelines...), nil
}
