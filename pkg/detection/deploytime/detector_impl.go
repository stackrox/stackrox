package deploytime

import (
	"context"

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
func (d *detectorImpl) Detect(ctx context.Context, enhancedDeployment booleanpolicy.EnhancedDeployment, opts ...DetectOption) ([]*storage.Alert, error) {
	var cfg detectionConfig
	for _, o := range opts {
		if o == nil {
			continue
		}
		o(&cfg)
	}

	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		for _, filter := range cfg.policyFilters {
			if !filter(compiled.Policy()) {
				return nil
			}
		}
		// Check predicate on deployment.
		if !compiled.AppliesTo(ctx, enhancedDeployment.Deployment) {
			return nil
		}

		// Check enforcement on deployment if we don't want unenforced alerts.
		enforcement, _ := buildEnforcement(compiled.Policy(), enhancedDeployment.Deployment)
		if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && cfg.enforcementOnly {
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
