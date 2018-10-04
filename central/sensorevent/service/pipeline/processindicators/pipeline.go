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
		return s.process(event.GetProcessIndicator())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *v1.ProcessIndicator) (*v1.SensorEnforcement, error) {
	err := s.indicators.AddProcessIndicator(indicator)
	if err != nil {
		return nil, err
	}
	deployment, exists, err := s.deploymentDataStore.GetDeployment(indicator.GetDeploymentId())
	if err != nil {
		return nil, fmt.Errorf("error getting deployment details from data store: %s", err)
	}
	if !exists {
		return nil, fmt.Errorf("couldn't find deployment details for indicator: %+v", indicator)
	}

	// populate process data
	indicators, err := s.indicators.SearchRawProcessIndicators(
		search.NewQueryBuilder().
			AddStrings(search.DeploymentID, deployment.GetId()).
			ProtoQuery(),
	)
	if err != nil {
		return nil, err
	}
	deployment.Processes = indicators

	return s.createResponse(deployment, indicator)
}

func (s *pipelineImpl) createResponse(deployment *v1.Deployment, indicator *v1.ProcessIndicator) (*v1.SensorEnforcement, error) {
	var sensorEnforcement *v1.SensorEnforcement
	newAlerts, err := s.detector.Detect(deployment)
	if err != nil {
		return sensorEnforcement, err
	}

	oldAlerts, err := s.alertManager.GetAlertsByDeploymentAndPolicyLifecycle(deployment.GetId(), v1.LifecycleStage_RUN_TIME)
	if err != nil {
		return sensorEnforcement, err
	}

	err = s.alertManager.AlertAndNotify(oldAlerts, newAlerts)
	if err != nil {
		return sensorEnforcement, err
	}

	return enforcementActions(deployment, newAlerts, indicator), nil
}

func enforcementActions(deployment *v1.Deployment, alerts []*v1.Alert, indicator *v1.ProcessIndicator) *v1.SensorEnforcement {
	if !killPodActionSupported(alerts, indicator) {
		return nil
	}

	return createEnforcementAction(deployment, indicator)
}

// killPodActionSupported returns true if the alert supports kill pod action.
func killPodActionSupported(alerts []*v1.Alert, indicator *v1.ProcessIndicator) bool {
	for _, alert := range alerts {
		violations := alert.GetViolations()
		for _, v := range violations {
			indicators := v.GetProcesses()
			for _, singleIndicator := range indicators {
				if singleIndicator.Id == indicator.Id {
					if alert.GetEnforcement().GetAction() == v1.EnforcementAction_KILL_POD_ENFORCEMENT {
						return true
					}
				}
			}
		}
	}
	return false
}

func createEnforcementAction(deployment *v1.Deployment, indicator *v1.ProcessIndicator) (sensorEnforcement *v1.SensorEnforcement) {
	containerID := ""
	if indicator.GetSignal() != nil {
		containerID = indicator.GetSignal().ContainerId
	}

	containers := deployment.GetContainers()
	for _, container := range containers {
		for _, instance := range container.GetInstances() {
			if containerID == instance.InstanceId.Id[:12] {
				resource := &v1.SensorEnforcement_ContainerInstance{
					ContainerInstance: &v1.ContainerInstanceEnforcement{
						ContainerInstanceId: instance.InstanceId.Id,
						PodId:               instance.ContainingPodId,
						Namespace:           deployment.Namespace,
					},
				}
				sensorEnforcement = &v1.SensorEnforcement{
					Enforcement: v1.EnforcementAction_KILL_POD_ENFORCEMENT,
					Resource:    resource,
				}
				return
			}
		}
	}
	return
}
