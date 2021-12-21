package deploytime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/logging"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx DetectionContext, deployment *storage.Deployment, images []*storage.Image, filters ...detection.FilterOption) ([]*storage.Alert, error) {
	log := logging.LoggerForModule()
	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	if deployment.GetName() == "vulnerable-deploy-enforce" {
		policyNames := "| "
		_ = d.policySet.ForEach(func(c detection.CompiledPolicy) error {
			policyNames = policyNames + c.Policy().GetName() + " || "
			return nil
		})
		//log.Errorf("Running deploy time detect for %s with policies %s", deployment.GetName(), policyNames)
		log.Errorf("Running deploy time detect for %s", deployment.GetName())
	}
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if deployment.GetName() == "vulnerable-deploy-enforce" {
			if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
				log.Errorf("[Detect] Deploy: %s, Policy: %s, Disabled: %s", deployment.GetName(), compiled.Policy().GetName(), compiled.Policy().GetDisabled())
			}
		}
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
			if deployment.GetName() == "vulnerable-deploy-enforce" {
				if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
					log.Errorf("[Detect] Policy doesn't apply to %s!?", deployment.GetName())
				}
			}
			return nil
		}

		// Check enforcement on deployment if we don't want unenforced alerts.
		enforcement, _ := buildEnforcement(compiled.Policy(), deployment)
		if deployment.GetName() == "vulnerable-deploy-enforce" {
			if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
				log.Errorf("[Detect] Enforcement: %s, Deploy: %s", enforcement.String(), deployment.GetName())
			}
		}
		if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && ctx.EnforcementOnly {
			if deployment.GetName() == "vulnerable-deploy-enforce" {
				if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
					log.Errorf("[Detect] Enforcement only but this isn't enforcing for %s!", deployment.GetName())
				}
			}
			return nil
		}

		// Generate violations.
		violations, err := compiled.MatchAgainstDeployment(&cacheReceptacle, deployment, images)
		if err != nil {
			return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), deployment.GetNamespace(), deployment.GetName())
		}
		if alertViolations := violations.AlertViolations; len(alertViolations) > 0 {
			if deployment.GetName() == "vulnerable-deploy-enforce" {
				if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
					log.Errorf("[Detect] Got %d violations for %s", len(alertViolations), deployment.GetName())
				}
			}
			alerts = append(alerts, PolicyDeploymentAndViolationsToAlert(compiled.Policy(), deployment, alertViolations))
		} else {
			if deployment.GetName() == "vulnerable-deploy-enforce" {
				if compiled.Policy().GetName() == "e2e-vuln-DEFERRED-enforce" || compiled.Policy().GetName() == "e2e-vuln-FALSE_POSITIVE-enforce" {
					log.Errorf("[Detect] Got %d violations for %s", len(violations.AlertViolations), deployment.GetName())
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}
