package runtime

import (
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/generated/api/v1"
	containerMatcher "github.com/stackrox/rox/pkg/compiledpolicies/container/matcher"
)

type detectorImpl struct {
	policySet PolicySet
}

// // Detect runs detection on a container, returning any generated alerts.
func (d *detectorImpl) Detect(container *v1.Container) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	d.policySet.ForEach(func(p *v1.Policy, matcher containerMatcher.Matcher) error {
		if violations := matcher(container); len(violations) > 0 {
			alerts = append(alerts, utils.PolicyAndViolationsToAlert(p, violations))
		}
		return nil
	})
	return alerts, nil
}
