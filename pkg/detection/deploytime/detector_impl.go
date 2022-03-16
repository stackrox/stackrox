package deploytime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/detection"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx DetectionContext, deployment *storage.Deployment, images []*storage.Image, netpol *augmentedobjs.NetworkPolicyAssociation, filters ...detection.FilterOption) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		for _, filter := range filters {
			if !filter(compiled.Policy()) {
				return nil
			}
		}
		// Check predicate on deployment.
		if !compiled.AppliesTo(deployment) {
			return nil
		}

		// Check enforcement on deployment if we don't want unenforced alerts.
		enforcement, _ := buildEnforcement(compiled.Policy(), deployment)
		if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && ctx.EnforcementOnly {
			return nil
		}

		// Generate violations.
		violations, err := compiled.MatchAgainstDeployment(&cacheReceptacle, deployment, images, netpol)
		if err != nil {
			return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), deployment.GetNamespace(), deployment.GetName())
		}
		if alertViolations := violations.AlertViolations; len(alertViolations) > 0 {
			alerts = append(alerts, PolicyDeploymentAndViolationsToAlert(compiled.Policy(), deployment, alertViolations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}
