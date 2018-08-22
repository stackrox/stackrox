package buildtime

import (
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/generated/api/v1"
	imageMatcher "github.com/stackrox/rox/pkg/compiledpolicies/image/matcher"
)

type detectorImpl struct {
	policySet PolicySet
}

// Detect runs detection on an image, returning any generated alerts.
func (d *detectorImpl) Detect(image *v1.Image) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	d.policySet.ForEach(func(p *v1.Policy, matcher imageMatcher.Matcher) error {
		if violations := matcher(image); len(violations) > 0 {
			alerts = append(alerts, utils.PolicyAndViolationsToAlert(p, violations))
		}
		return nil
	})
	return alerts, nil
}
