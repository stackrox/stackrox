package deploytime

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var logger = logging.LoggerForModule()

type detectorImpl struct {
	policySet deployment.PolicySet

	deployments deploymentDataStore.DataStore
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) UpsertPolicy(policy *v1.Policy) error {
	return d.policySet.UpsertPolicy(policy)
}

// RemovePolicy removes a policy from the set.
func (d *detectorImpl) RemovePolicy(policyID string) error {
	return d.policySet.RemovePolicy(policyID)

}

func (d *detectorImpl) AlertsForDeployment(deployment *v1.Deployment) ([]*v1.Alert, error) {
	// Get the new and old alerts for the deployment.
	var newAlerts []*v1.Alert
	err := d.policySet.ForEach(func(p *v1.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		if shouldProcess != nil && !shouldProcess(deployment) {
			return nil
		}

		violations, err := matcher.MatchOne(d.deployments, deployment.GetId())
		if err != nil {
			return fmt.Errorf("evaluating violations for policy %s; deployment %s/%s: %s", p.GetName(), deployment.GetNamespace(), deployment.GetName(), err)
		}

		if len(violations) > 0 {
			newAlerts = append(newAlerts, policyDeploymentAndViolationsToAlert(p, deployment, violations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newAlerts, nil
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*v1.Alert, error) {
	var newAlerts []*v1.Alert
	err := d.policySet.ForOne(policyID, func(p *v1.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		violationsByDeployment, err := matcher.Match(d.deployments)
		if err != nil {
			return err
		}
		for deploymentID, violations := range violationsByDeployment {
			dep, exists, err := d.deployments.GetDeployment(deploymentID)
			if err != nil {
				return err
			}
			if !exists {
				logger.Errorf("deployment with id '%s' had violations, but doesn't exist", deploymentID)
				continue
			}
			if shouldProcess != nil && !shouldProcess(dep) {
				continue
			}
			newAlerts = append(newAlerts, policyDeploymentAndViolationsToAlert(p, dep, violations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newAlerts, nil
}

// policyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *v1.Policy, deployment *v1.Deployment, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: v1.LifecycleStage_DEPLOY,
		Deployment:     proto.Clone(deployment).(*v1.Deployment),
		Policy:         proto.Clone(policy).(*v1.Policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := policyAndDeploymentToEnforcement(policy, deployment); action != v1.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &v1.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
		logger.Warnf("generated deploy alert with enforcement for deployment %s: %s", alert.GetDeployment().GetName(), proto.MarshalTextString(alert.GetEnforcement()))
	}
	return alert
}

// policyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func policyAndDeploymentToEnforcement(policy *v1.Policy, deployment *v1.Deployment) (enforcement v1.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT && scaleToZeroEnabled(deployment) {
			return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
		}
		if enforcementAction == v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
			return v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, fmt.Sprintf("Unsatisfiable node constraint applied to deployment %s", deployment.GetName())
		}
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, ""
}

const (
	globalDeployment    = "Global"
	daemonSetDeployment = "DaemonSet"
)

func scaleToZeroEnabled(deployment *v1.Deployment) bool {
	if deployment.GetType() == globalDeployment || deployment.GetType() == daemonSetDeployment {
		return false
	}
	return true
}
