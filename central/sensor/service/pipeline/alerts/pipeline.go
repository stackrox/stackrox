package alerts

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
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
	for _, a := range alertResults.GetAlerts() {
		a.Deployment.ClusterId = clusterID
		a.Deployment.ClusterName = clusterName
	}
	if err := s.lifecycleManager.HandleAlerts(alertResults.GetDeploymentId(), alertResults.GetAlerts(), alertResults.GetStage()); err != nil {
		return errors.Wrap(err, "error handling alerts")
	}

	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
