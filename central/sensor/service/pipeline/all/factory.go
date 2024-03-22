package all

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/pipelines/complianceoperatorprofiles"
	"github.com/stackrox/rox/central/complianceoperator/pipelines/complianceoperatorresults"
	"github.com/stackrox/rox/central/complianceoperator/pipelines/complianceoperatorrules"
	"github.com/stackrox/rox/central/complianceoperator/pipelines/complianceoperatorscans"
	"github.com/stackrox/rox/central/complianceoperator/pipelines/complianceoperatorscansettingbinding"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorinfo"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorprofilesv2"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorresultsv2"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorrulesv2"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorscansettingbindingsv2"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorscansv2"
	"github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorsuitesv2"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/alerts"
	"github.com/stackrox/rox/central/sensor/service/pipeline/auditlogstateupdate"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clusterhealthupdate"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clusterstatusupdate"
	"github.com/stackrox/rox/central/sensor/service/pipeline/deploymentevents"
	"github.com/stackrox/rox/central/sensor/service/pipeline/enhancements"
	"github.com/stackrox/rox/central/sensor/service/pipeline/imageintegrations"
	"github.com/stackrox/rox/central/sensor/service/pipeline/namespaces"
	"github.com/stackrox/rox/central/sensor/service/pipeline/networkflowupdate"
	"github.com/stackrox/rox/central/sensor/service/pipeline/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodeinventory"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/rox/central/sensor/service/pipeline/podevents"
	"github.com/stackrox/rox/central/sensor/service/pipeline/processindicators"
	"github.com/stackrox/rox/central/sensor/service/pipeline/processlisteningonport"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reprocessing"
	"github.com/stackrox/rox/central/sensor/service/pipeline/rolebindings"
	"github.com/stackrox/rox/central/sensor/service/pipeline/roles"
	"github.com/stackrox/rox/central/sensor/service/pipeline/secrets"
	"github.com/stackrox/rox/central/sensor/service/pipeline/serviceaccounts"
)

// NewFactory returns a new instance of a Factory that produces a pipeline handling all message types.
func NewFactory(manager hashManager.Manager) pipeline.Factory {
	return &factoryImpl{
		manager: manager,
	}
}

type factoryImpl struct {
	manager hashManager.Manager
}

// PipelineForCluster grabs items from the queue, processes them, and potentially sends them back to sensor.
func (s *factoryImpl) PipelineForCluster(ctx context.Context, clusterID string) (pipeline.ClusterPipeline, error) {
	flowUpdateFragment, err := networkflowupdate.Singleton().GetFragment(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pipelines := []pipeline.Fragment{
		deploymentevents.GetPipeline(),
		podevents.GetPipeline(),
		processindicators.GetPipeline(),
		processlisteningonport.GetPipeline(),
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
		complianceoperatorresults.GetPipeline(),
		complianceoperatorprofiles.GetPipeline(),
		complianceoperatorscansettingbinding.GetPipeline(),
		complianceoperatorrules.GetPipeline(),
		complianceoperatorscans.GetPipeline(),
		nodeinventory.GetPipeline(),
		enhancements.GetPipeline(),
		complianceoperatorinfo.GetPipeline(),
		complianceoperatorrulesv2.GetPipeline(),
		complianceoperatorresultsv2.GetPipeline(),
		complianceoperatorprofilesv2.GetPipeline(),
		complianceoperatorscansv2.GetPipeline(),
		complianceoperatorscansettingbindingsv2.GetPipeline(),
		complianceoperatorsuitesv2.GetPipeline(),
	}

	deduper := s.manager.GetDeduper(ctx, clusterID)
	return NewClusterPipeline(clusterID, deduper, pipelines...), nil
}
