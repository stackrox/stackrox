package runtime

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
	"github.com/stackrox/rox/pkg/uuid"
)

type detectorImpl struct {
	policySet deployment.PolicySet
}

// // Detect runs detection on a container, returning any generated alerts.
func (d *detectorImpl) Detect(deployment *v1.Deployment) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	d.policySet.ForEach(func(p *v1.Policy, matcher deploymentMatcher.Matcher) error {
		if violations := matcher(deployment); len(violations) > 0 {
			alerts = append(alerts, policyDeploymentAndViolationsToAlert(p, deployment, violations))
		}
		return nil
	})
	return alerts, nil
}

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *v1.Policy, deployment *v1.Deployment, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: v1.LifecycleStage_RUN_TIME,
		Deployment:     proto.Clone(deployment).(*v1.Deployment),
		Policy:         proto.Clone(policy).(*v1.Policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := PolicyAndDeploymentToEnforcement(policy, deployment); action != v1.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &v1.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// PolicyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func PolicyAndDeploymentToEnforcement(policy *v1.Policy, deployment *v1.Deployment) (enforcement v1.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == v1.EnforcementAction_KILL_POD_ENFORCEMENT {
			return v1.EnforcementAction_KILL_POD_ENFORCEMENT, fmt.Sprintf("Deployment %s has pods killed in response to policy violation", deployment.GetName())
		}
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, ""
}
