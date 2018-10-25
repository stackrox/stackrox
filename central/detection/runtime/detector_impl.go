package runtime

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/uuid"
)

type detectorImpl struct {
	policySet   deployment.PolicySet
	deployments datastore.DataStore
}

type alertSlice struct {
	alerts []*v1.Alert
}

func (a *alertSlice) append(alerts ...*v1.Alert) {
	a.alerts = append(a.alerts, alerts...)
}

func (d *detectorImpl) policyMatcher(alerts *alertSlice, deploymentIDs ...string) func(*v1.Policy, searchbasedpolicies.Matcher, predicate.Predicate) error {
	return func(p *v1.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		var err error
		var violationsByDeployment map[string][]*v1.Alert_Violation
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
				logger.Errorf("deployment with id '%s' had violations, but doesn't exist", deploymentID)
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

func (d *detectorImpl) AlertsForDeployments(deploymentIDs ...string) ([]*v1.Alert, error) {
	alertSlice := &alertSlice{}
	err := d.policySet.ForEach(d.policyMatcher(alertSlice, deploymentIDs...))
	if err != nil {
		return nil, err
	}
	return alertSlice.alerts, nil
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*v1.Alert, error) {
	alertSlice := &alertSlice{}
	err := d.policySet.ForOne(policyID, d.policyMatcher(alertSlice))
	if err != nil {
		return nil, err
	}
	return alertSlice.alerts, nil
}

func (d *detectorImpl) UpsertPolicy(policy *v1.Policy) error {
	return d.policySet.UpsertPolicy(policy)
}

func (d *detectorImpl) RemovePolicy(policyID string) error {
	return d.policySet.RemovePolicy(policyID)
}

func (d *detectorImpl) AlertsForDeployment(deployment *v1.Deployment) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	err := d.policySet.ForEach(func(p *v1.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		if shouldProcess != nil && !shouldProcess(deployment) {
			return nil
		}
		violations, err := matcher.MatchOne(d.deployments, deployment.GetId())
		if err != nil {
			return fmt.Errorf("matching against deployment %s/%s: %s", deployment.GetNamespace(), deployment.GetName(), err)
		}
		if len(violations) > 0 {
			alerts = append(alerts, policyDeploymentAndViolationsToAlert(p, deployment, violations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *v1.Policy, deployment *v1.Deployment, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: v1.LifecycleStage_RUNTIME,
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
	}
	return alert
}

// policyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func policyAndDeploymentToEnforcement(policy *v1.Policy, deployment *v1.Deployment) (enforcement v1.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == v1.EnforcementAction_KILL_POD_ENFORCEMENT {
			return v1.EnforcementAction_KILL_POD_ENFORCEMENT, fmt.Sprintf("Deployment %s has pods killed in response to policy violation", deployment.GetName())
		}
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, ""
}
