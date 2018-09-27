package runtime

import (
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/generated/api/v1"
	deploymentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/deployment/matcher"
)

type detectorImpl struct {
	policySet deployment.PolicySet
}

// // Detect runs detection on a container, returning any generated alerts.
func (d *detectorImpl) Detect(deployment *v1.Deployment) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	d.policySet.ForEach(func(p *v1.Policy, matcher deploymentMatcher.Matcher) error {
		if violations := matcher(deployment); len(violations) > 0 {
			alerts = append(alerts, utils.PolicyAndViolationsToAlert(p, violations))
		}
		return nil
	})
	return alerts, nil
}
