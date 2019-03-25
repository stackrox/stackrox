package runtime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

type detectorImpl struct {
	policySet   deployment.PolicySet
	deployments datastore.DataStore
}

type alertSlice struct {
	alerts []*storage.Alert
}

func (a *alertSlice) append(alerts ...*storage.Alert) {
	a.alerts = append(a.alerts, alerts...)
}

func (d *detectorImpl) policyMatcher(alerts *alertSlice, deploymentIDs ...string) func(*storage.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error {
	return func(p *storage.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		var err error
		var violationsByDeployment map[string]searchbasedpolicies.Violations
		if len(deploymentIDs) == 0 {
			violationsByDeployment, err = matcher.Match(d.deployments)
		} else {
			violationsByDeployment, err = matcher.MatchMany(d.deployments, deploymentIDs...)
		}

		if err != nil {
			return fmt.Errorf("matching policy %s: %s", p.GetName(), err)
		}

		for deploymentID, violations := range violationsByDeployment {
			dep, exists, err := d.deployments.GetDeployment(deploymentID)
			if err != nil {
				return err
			}
			if !exists {
				log.Errorf("deployment with id '%s' had violations, but doesn't exist", deploymentID)
				continue
			}
			if shouldProcess != nil && !shouldProcess(dep) {
				continue
			}
			alerts.append(policyDeploymentAndViolationsToAlert(p, dep, violations))
		}
		return nil
	}
}

func (d *detectorImpl) AlertsForDeployments(deploymentIDs ...string) ([]*storage.Alert, error) {
	alertSlice := &alertSlice{}
	err := d.policySet.ForEach(d.policyMatcher(alertSlice, deploymentIDs...))
	if err != nil {
		return nil, err
	}
	return alertSlice.alerts, nil
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*storage.Alert, error) {
	alertSlice := &alertSlice{}
	err := d.policySet.ForOne(policyID, d.policyMatcher(alertSlice))
	if err != nil {
		return nil, err
	}
	return alertSlice.alerts, nil
}

func (d *detectorImpl) DeploymentWhitelistedForPolicy(deploymentID, policyID string) (isWhitelisted bool) {
	err := d.policySet.ForOne(policyID, func(p *storage.Policy, _ searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		if p.GetDisabled() {
			isWhitelisted = true
			return nil
		}
		dep, exists, err := d.deployments.GetDeployment(deploymentID)
		if err != nil {
			return err
		}
		if !exists {
			// Assume it's not whitelisted if it doesn't exist, otherwise runtime alerts for deleted deployments
			// will always get removed every time we update a policy.
			isWhitelisted = false
			return nil
		}
		if shouldProcess == nil {
			isWhitelisted = false
			return nil
		}
		isWhitelisted = !shouldProcess(dep)
		return nil
	})
	if err != nil {
		log.Errorf("Couldn't evaluate whitelist for deployment %s, policy %s", deploymentID, policyID)
	}
	return
}

func (d *detectorImpl) UpsertPolicy(policy *storage.Policy) error {
	return d.policySet.UpsertPolicy(policy)
}

func (d *detectorImpl) RemovePolicy(policyID string) error {
	return d.policySet.RemovePolicy(policyID)
}

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations searchbasedpolicies.Violations) *storage.Alert {
	if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
		return nil
	}
	alert := &storage.Alert{
		Id:               uuid.NewV4().String(),
		LifecycleStage:   storage.LifecycleStage_RUNTIME,
		Deployment:       protoutils.CloneStorageDeployment(deployment),
		Policy:           protoutils.CloneStoragePolicy(policy),
		Violations:       violations.AlertViolations,
		ProcessViolation: violations.ProcessViolation,
		Time:             ptypes.TimestampNow(),
	}
	if action, msg := policyAndDeploymentToEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// policyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func policyAndDeploymentToEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			return storage.EnforcementAction_KILL_POD_ENFORCEMENT, fmt.Sprintf("Deployment %s has pods killed in response to policy violation", deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}
