package deploytime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var logger = logging.LoggerForModule()

type detectorImpl struct {
	policySet deployment.PolicySet

	deployments deploymentDataStore.DataStore
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) UpsertPolicy(policy *storage.Policy) error {
	return d.policySet.UpsertPolicy(policy)
}

// RemovePolicy removes a policy from the set.
func (d *detectorImpl) RemovePolicy(policyID string) error {
	return d.policySet.RemovePolicy(policyID)

}

func (d *detectorImpl) AlertsForDeployment(deployment *storage.Deployment) ([]*v1.Alert, error) {
	// Get the new and old alerts for the deployment.
	var newAlerts []*v1.Alert
	err := d.policySet.ForEach(func(p *storage.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
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
	err := d.policySet.ForOne(policyID, func(p *storage.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
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
func policyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		Deployment:     protoutils.CloneStorageDeployment(deployment),
		Policy:         protoutils.CloneStoragePolicy(policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := policyAndDeploymentToEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &v1.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// policyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func policyAndDeploymentToEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
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
