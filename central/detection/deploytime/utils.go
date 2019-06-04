package deploytime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/convert"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// Expansion of executor that holds resulting alerts.
type alertCollectingExecutor interface {
	detection.PolicyExecutor

	GetAlerts() []*storage.Alert
	ClearAlerts()
}

// policyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations []*storage.Alert_Violation) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}

	alertDeployment := convert.ToAlertDeployment(deployment)

	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		Deployment:     alertDeployment,
		Policy:         protoutils.CloneStoragePolicy(policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := buildEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// buildEnforcement returns the enforcement action and message for deploy time enforcment.
func buildEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT && scaleToZeroEnabled(deployment) {
			return storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
		}
		if enforcementAction == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
			return storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, fmt.Sprintf("Unsatisfiable node constraint applied to deployment %s", deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}

const (
	globalDeployment    = "Global"
	daemonSetDeployment = "DaemonSet"
)

func scaleToZeroEnabled(deployment *storage.Deployment) bool {
	if deployment.GetType() == globalDeployment || deployment.GetType() == daemonSetDeployment {
		return false
	}
	return true
}
