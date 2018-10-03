package processindicators

import (
	"fmt"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators datastore.DataStore, detector runtime.Detector,
	alertManager utils.AlertManager, deploymentStore deploymentDataStore.DataStore) pipeline.Pipeline {
	return &pipelineImpl{
		indicators:          indicators,
		detector:            detector,
		alertManager:        alertManager,
		deploymentDataStore: deploymentStore,
	}
}

type pipelineImpl struct {
	indicators          datastore.DataStore
	detector            runtime.Detector
	alertManager        utils.AlertManager
	deploymentDataStore deploymentDataStore.DataStore
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error) {
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return nil, s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return nil, s.process(event.GetProcessIndicator())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *v1.ProcessIndicator) error {
	err := s.indicators.AddProcessIndicator(indicator)
	if err != nil {
		return err
	}
	deployment, exists, err := s.deploymentDataStore.GetDeployment(indicator.GetDeploymentId())
	if err != nil {
		return fmt.Errorf("error getting deployment details from data store: %s", err)
	}
	if !exists {
		return fmt.Errorf("couldn't find deployment details for indicator: %+v", indicator)
	}

	// populate process data
	indicators, err := s.indicators.SearchRawProcessIndicators(
		search.NewQueryBuilder().
			AddStrings(search.DeploymentID, deployment.GetId()).
			ProtoQuery(),
	)
	if err != nil {
		return err
	}
	deployment.Processes = indicators
	log.Debugf("Processed indicators for deployment %s: %v", deployment.GetId(), deployment.Processes)
	return s.reconcileAlerts(deployment)
}

func (s *pipelineImpl) reconcileAlerts(deployment *v1.Deployment) error {
	newAlerts, err := s.detector.Detect(deployment)
	if err != nil {
		return err
	}

	oldAlerts, err := s.alertManager.GetAlertsByDeploymentAndPolicyLifecycle(deployment.GetId(), v1.LifecycleStage_RUN_TIME)
	if err != nil {
		return err
	}

	return s.alertManager.AlertAndNotify(oldAlerts, newAlerts)
}
