package reprocessing

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichanddetect"
	countMetrics "github.com/stackrox/rox/central/metrics"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
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

func (s *pipelineImpl) Reconcile(clusterID string) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetReprocessDeployments() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.DeploymentReprocess)

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).AddExactMatches(search.DeploymentID, msg.GetReprocessDeployments().GetDeploymentIds()...).ProtoQuery()
	deployments, err := s.deployments.SearchRawDeployments(q)
	if err != nil {
		return err
	}

	switch msg.GetReprocessDeployments().Target.(type) {
	case *central.ReprocessDeployments_Risk:
		for _, d := range deployments {
			s.riskManager.ReprocessDeploymentRisk(d)
		}
	case *central.ReprocessDeployments_All:
		for _, d := range deployments {
			if err := s.manager.EnrichAndDetect(d); err != nil {
				log.Error(err)
			}
		}
	}
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
