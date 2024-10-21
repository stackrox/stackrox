package deploytime

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// PolicySet returns set of policies.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on a deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx DetectionContext, enhancedDeployment booleanpolicy.EnhancedDeployment, filters ...detection.FilterOption) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		log.Debugf("detectorImpl: Detect: checking policy %s for deployment %s", compiled.Policy().GetName(), enhancedDeployment.Deployment.GetName())

		for _, filter := range filters {
			if !filter(compiled.Policy()) {
				log.Debugf("detectorImpl: Detect: policy %s for deployment %s - exiting due to filter", compiled.Policy().GetName(), enhancedDeployment.Deployment.GetName())
				return nil
			}
		}
		// Check predicate on deployment.
		if !compiled.AppliesTo(enhancedDeployment.Deployment) {
			log.Debugf("detectorImpl: Detect: policy %s for deployment %s - policy does not apply", compiled.Policy().GetName(), enhancedDeployment.Deployment.GetName())
			return nil
		}

		// Check enforcement on deployment if we don't want unenforced alerts.
		enforcement, _ := buildEnforcement(compiled.Policy(), enhancedDeployment.Deployment)
		if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && ctx.EnforcementOnly {
			return nil
		}

		// Generate violations.
		violations, err := compiled.MatchAgainstDeployment(&cacheReceptacle, enhancedDeployment)
		if err != nil {
			return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), enhancedDeployment.Deployment.GetNamespace(), enhancedDeployment.Deployment.GetName())
		}
		if alertViolations := violations.AlertViolations; len(alertViolations) > 0 {
			alerts = append(alerts, PolicyDeploymentAndViolationsToAlert(compiled.Policy(), enhancedDeployment.Deployment, alertViolations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}
