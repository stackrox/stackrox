package reprocessing

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichanddetect"
	countMetrics "github.com/stackrox/rox/central/metrics"
	riskManager "github.com/stackrox/rox/central/risk/manager"
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
	return NewPipeline(datastore.Singleton(), enrichanddetect.Singleton(), riskManager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(deployments datastore.DataStore, manager enrichanddetect.EnricherAndDetector, riskManager riskManager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		riskManager: riskManager,
		manager:     manager,
		deployments: deployments,
	}
}

type pipelineImpl struct {
	deployments datastore.DataStore
	riskManager riskManager.Manager
	manager     enrichanddetect.EnricherAndDetector
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, _ *reconciliation.StoreMap) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetReprocessDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.DeploymentReprocess)

	reprocessMsg := msg.GetEvent().GetReprocessDeployment()

	deployment, exists, err := s.deployments.GetDeployment(ctx, reprocessMsg.GetDeploymentId())
	if err != nil || !exists {
		return err
	}

	if reprocessMsg.RiskOnly {
		s.riskManager.ReprocessDeploymentRisk(deployment)
	} else {
		return s.manager.EnrichAndDetect(deployment)
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
