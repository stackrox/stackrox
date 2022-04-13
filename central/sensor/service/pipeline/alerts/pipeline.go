package alerts

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), deploymentDataStore.Singleton(), lifecycle.SingletonManager())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, manager lifecycle.Manager) pipeline.Fragment {
	return &pipelineImpl{
		lifecycleManager: manager,
		clusters:         clusters,
		deployments:      deployments,
	}
}

type pipelineImpl struct {
	lifecycleManager lifecycle.Manager
	clusters         clusterDataStore.DataStore
	deployments      deploymentDataStore.DataStore
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetAlertResults() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Alert)

	clusterName, exists, err := s.clusters.GetClusterName(ctx, clusterID)
	if err != nil {
		return errors.Wrap(err, "error getting cluster name")
	}
	if !exists {
		return nil
	}

	alertResults := msg.GetEvent().GetAlertResults()
	if msg.GetEvent().GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		if len(alertResults.GetAlerts()) > 0 {
			return errors.Errorf("unexpected: Got non-zero alerts for a deployment remove: %+v", msg.GetEvent())
		}
		if err := s.lifecycleManager.DeploymentRemoved(alertResults.GetDeploymentId()); err != nil {
			return err
		}

		return nil
	}

	for _, a := range alertResults.GetAlerts() {
		if deployment := a.GetDeployment(); deployment != nil {
			deployment.ClusterId = clusterID
			deployment.ClusterName = clusterName
		}
		if resource := a.GetResource(); resource != nil {
			resource.ClusterId = clusterID
			resource.ClusterName = clusterName
		}
	}

	// All alerts in an `alertResults` message will correspond to just one source (ie, either audit event or deployment), by construction.
	if alertResults.GetSource() == central.AlertResults_AUDIT_EVENT {
		if err := s.lifecycleManager.HandleResourceAlerts(clusterID, alertResults.GetAlerts(), alertResults.GetStage()); err != nil {
			return errors.Wrap(err, "error handling resource alerts")
		}
		return nil
	}

	// Treat all other alerts, even if they don't have a listed deployment as a "non-resource" alert for backwards compatibility
	if err := s.lifecycleManager.HandleDeploymentAlerts(alertResults.GetDeploymentId(), alertResults.GetAlerts(), alertResults.GetStage()); err != nil {
		return errors.Wrap(err, "error handling alerts")
	}

	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
