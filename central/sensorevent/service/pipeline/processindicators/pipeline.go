package processindicators

import (
	"fmt"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators datastore.DataStore, policySet deployTimeDetection.PolicySet,
	alertManager deployTimeDetection.AlertManager, deploymentStore deploymentDataStore.DataStore) pipeline.Pipeline {
	return &pipelineImpl{
		indicators:          indicators,
		policySet:           policySet,
		alertManager:        alertManager,
		deploymentDataStore: deploymentStore,
	}
}

type pipelineImpl struct {
	indicators          datastore.DataStore
	policySet           deployTimeDetection.PolicySet
	alertManager        deployTimeDetection.AlertManager
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
	return s.reconcileAlerts(deployment, indicators)
}

func (s *pipelineImpl) reconcileAlerts(deployment *v1.Deployment, indicators []*v1.ProcessIndicator) error {
	previousAlerts, err := s.alertManager.GetAlertsByDeployment(deployment.GetId())
	if err != nil {
		return err
	}

	var newAlerts []*v1.Alert
	s.policySet.ForEach(func(p *v1.Policy, matcher deploymentMatcher.Matcher) error {
		if violations := matcher(deployment); len(violations) > 0 {
			newAlerts = append(newAlerts, utils.PolicyDeploymentAndViolationsToAlert(p, deployment, violations))
		}
		return nil
	}, true)

	if len(previousAlerts) != len(newAlerts) {
		err := s.alertManager.AlertAndNotify(previousAlerts, newAlerts)
		if err != nil {
			return err
		}
		log.Infof("Added new alert!")
	}
	return nil
}
