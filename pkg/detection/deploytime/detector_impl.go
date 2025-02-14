package deploytime

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type detectorImpl struct {
	policySet detection.PolicySet
}

func filterLogging(deploymentName string, fmt string, args ...interface{}) {
	if strings.Contains(deploymentName, "test") {
		log.Infof(fmt, args...)
	}
}

// PolicySet returns set of policies.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on a deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx DetectionContext, enhancedDeployment booleanpolicy.EnhancedDeployment, filters ...detection.FilterOption) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	deploymentName := enhancedDeployment.Deployment.GetName()
	defer filterLogging(deploymentName, "lvm --> deployment %s processed", deploymentName)
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		policyName := compiled.Policy().GetName()
		filterLogging(deploymentName, "lvm --> Policy Name %s", policyName)
		if compiled.Policy().GetDisabled() {
			filterLogging(deploymentName, "lvm --> Policy %s disabled", policyName)
			return nil
		}
		for _, filter := range filters {
			if !filter(compiled.Policy()) {
				filterLogging(deploymentName, "lvm --> Policy %s filtered", policyName)
				return nil
			}
		}
		// Check predicate on deployment.
		if !compiled.AppliesTo(enhancedDeployment.Deployment) {
			filterLogging(deploymentName, "lvm --> Policy %s does not apply", policyName)
			return nil
		}

		// Check enforcement on deployment if we don't want unenforced alerts.
		enforcement, _ := buildEnforcement(compiled.Policy(), enhancedDeployment.Deployment)
		if enforcement == storage.EnforcementAction_UNSET_ENFORCEMENT && ctx.EnforcementOnly {
			filterLogging(deploymentName, "lvm --> Policy %s enforcement", policyName)
			return nil
		}

		// Generate violations.
		violations, err := compiled.MatchAgainstDeployment(&cacheReceptacle, enhancedDeployment)
		if err != nil {
			filterLogging(deploymentName, "lvm --> Policy %s error matching deployment %v", policyName, err)
			return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), enhancedDeployment.Deployment.GetNamespace(), enhancedDeployment.Deployment.GetName())
		}
		filterLogging(deploymentName, "lvm --> Policy %s generated %d alerts", policyName, len(violations.AlertViolations))
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
