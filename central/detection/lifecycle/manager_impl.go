package lifecycle

import (
	"fmt"

	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/policies"
)

type managerImpl struct {
	enricher           enrichment.Enricher
	runtimeDetector    runtime.Detector
	deploytimeDetector deploytime.Detector
	alertManager       utils.AlertManager
}

func (m *managerImpl) IndicatorAdded(indicator *v1.ProcessIndicator, deployment *v1.Deployment) (*v1.SensorEnforcement, error) {
	newAlerts, err := m.runtimeDetector.AlertsForDeployment(deployment)
	if err != nil {
		return nil, err
	}

	oldAlerts, err := m.alertManager.GetAlertsByDeploymentAndPolicyLifecycle(indicator.GetDeploymentId(), v1.LifecycleStage_RUNTIME)
	if err != nil {
		return nil, fmt.Errorf("retrieving old alerts for deployment %s: %s", indicator.GetDeploymentId(), err)
	}

	err = m.alertManager.AlertAndNotify(oldAlerts, newAlerts)
	if err != nil {
		return nil, err
	}
	return enforcementActionForAddedIndicator(deployment, newAlerts, indicator), nil
}

func (m *managerImpl) DeploymentUpdated(deployment *v1.Deployment) (string, v1.EnforcementAction, error) {
	// Attempt to enrich the image before detection.
	if _, err := m.enricher.Enrich(deployment); err != nil {
		logger.Errorf("Error enriching deployment %s: %s", deployment.GetName(), err)
	}

	// Asynchronously update risk after processing.
	defer m.enricher.ReprocessDeploymentRiskAsync(deployment)

	// Get alerts for the new deployment from the current set of policies.
	var presentAlerts []*v1.Alert

	deployTimeAlerts, err := m.deploytimeDetector.AlertsForDeployment(deployment)
	if err != nil {
		logger.Errorf("Error fetching deploy time alerts: %s", err)
	} else {
		presentAlerts = append(presentAlerts, deployTimeAlerts...)
	}

	runTimeAlerts, err := m.runtimeDetector.AlertsForDeployment(deployment)
	if err != nil {
		logger.Errorf("Error fetching run time alerts: %s", err)
	} else {
		presentAlerts = append(presentAlerts, runTimeAlerts...)
	}

	// Get the previous alerts for the deployment (if any exist).
	previousAlerts, err := m.alertManager.GetAlertsByDeployment(deployment.GetId())
	if err != nil {
		return "", v1.EnforcementAction_UNSET_ENFORCEMENT, err
	}

	// Perform notifications and update DB.
	if err := m.alertManager.AlertAndNotify(previousAlerts, presentAlerts); err != nil {
		return "", v1.EnforcementAction_UNSET_ENFORCEMENT, err
	}

	// Generate enforcement actions based on the currently generated alerts.
	alertToEnforce, enforcementAction := determineEnforcement(presentAlerts)
	return alertToEnforce, enforcementAction, nil
}

func (m *managerImpl) UpsertPolicy(policy *v1.Policy) error {
	// Asynchronously update all deployments' risk after processing.
	defer m.enricher.ReprocessRiskAsync()

	var presentAlerts []*v1.Alert

	// Add policy to set.
	if policies.AppliesAtDeployTime(policy) {
		if err := m.deploytimeDetector.UpsertPolicy(policy); err != nil {
			return fmt.Errorf("adding policy %s to deploy time detector: %s", policy.GetName(), err)
		}
		deployTimeAlerts, err := m.deploytimeDetector.AlertsForPolicy(policy.GetId())
		if err != nil {
			return fmt.Errorf("error generating deploy-time alerts for policy %s: %s", policy.GetName(), err)
		}
		presentAlerts = append(presentAlerts, deployTimeAlerts...)
	} else {
		err := m.deploytimeDetector.RemovePolicy(policy.GetId())
		if err != nil {
			return fmt.Errorf("removing policy %s from deploy time detector: %s", policy.GetName(), err)
		}
	}

	if policies.AppliesAtRunTime(policy) {
		if err := m.runtimeDetector.UpsertPolicy(policy); err != nil {
			return fmt.Errorf("adding policy %s to runtime detector: %s", policy.GetName(), err)
		}
		runTimeAlerts, err := m.runtimeDetector.AlertsForPolicy(policy.GetId())
		if err != nil {
			return fmt.Errorf("error generating runtime alerts for policy %s: %s", policy.GetName(), err)
		}
		presentAlerts = append(presentAlerts, runTimeAlerts...)
	} else {
		err := m.runtimeDetector.RemovePolicy(policy.GetId())
		if err != nil {
			return fmt.Errorf("removing policy %s from runtime detector: %s", policy.GetName(), err)
		}
	}

	// Get any alerts previously existing for the policy (if any exist).
	previousAlerts, err := m.alertManager.GetAlertsByPolicy(policy.GetId())
	if err != nil {
		return err
	}

	// Perform notifications and update DB.
	return m.alertManager.AlertAndNotify(previousAlerts, presentAlerts)
}

func (m *managerImpl) DeploymentRemoved(deployment *v1.Deployment) error {
	oldAlerts, err := m.alertManager.GetAlertsByDeployment(deployment.GetId())
	if err != nil {
		return err
	}
	return m.alertManager.AlertAndNotify(oldAlerts, nil)
}

func (m *managerImpl) RemovePolicy(policyID string) error {
	if err := m.deploytimeDetector.RemovePolicy(policyID); err != nil {
		return err
	}
	if err := m.runtimeDetector.RemovePolicy(policyID); err != nil {
		return err
	}
	oldAlerts, err := m.alertManager.GetAlertsByPolicy(policyID)
	if err != nil {
		return err
	}
	return m.alertManager.AlertAndNotify(oldAlerts, nil)
}

// determineEnforcement returns the alert and its enforcement action to use from the input list (if any have enforcement).
func determineEnforcement(alerts []*v1.Alert) (alertID string, action v1.EnforcementAction) {
	for _, alert := range alerts {
		if alert.GetEnforcement().GetAction() == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return alert.GetId(), v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}

		if alert.GetEnforcement().GetAction() != v1.EnforcementAction_UNSET_ENFORCEMENT {
			alertID = alert.GetId()
			action = alert.GetEnforcement().GetAction()
		}
	}
	return
}

func enforcementActionForAddedIndicator(deployment *v1.Deployment, alerts []*v1.Alert, indicator *v1.ProcessIndicator) *v1.SensorEnforcement {
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
				if singleIndicator.GetId() == indicator.GetId() {
					if alert.GetEnforcement().GetAction() == v1.EnforcementAction_KILL_POD_ENFORCEMENT {
						return true
					}
				}
			}
		}
	}
	return false
}

func createEnforcementAction(deployment *v1.Deployment, indicator *v1.ProcessIndicator) *v1.SensorEnforcement {
	containerID := indicator.GetSignal().GetContainerId()
	if containerID == "" {
		return nil
	}

	containers := deployment.GetContainers()
	for _, container := range containers {
		for _, instance := range container.GetInstances() {
			if len(instance.GetInstanceId().GetId()) < 12 {
				continue
			}
			if containerID == instance.GetInstanceId().GetId()[:12] {
				resource := &v1.SensorEnforcement_ContainerInstance{
					ContainerInstance: &v1.ContainerInstanceEnforcement{
						ContainerInstanceId: instance.GetInstanceId().GetId(),
						PodId:               instance.GetContainingPodId(),
					},
				}
				return &v1.SensorEnforcement{
					Enforcement: v1.EnforcementAction_KILL_POD_ENFORCEMENT,
					Resource:    resource,
				}
			}
		}
	}
	return nil
}
