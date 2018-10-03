package buildtime

import (
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/generated/api/v1"
	imageMatcher "github.com/stackrox/rox/pkg/compiledpolicies/image/matcher"
	"github.com/stackrox/rox/pkg/uuid"
)

type detectorImpl struct {
	policySet image.PolicySet
}

// Detect runs detection on an image, returning any generated alerts.
func (d *detectorImpl) Detect(image *v1.Image) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	d.policySet.ForEach(func(p *v1.Policy, matcher imageMatcher.Matcher) error {
		if violations := matcher(image); len(violations) > 0 {
			alerts = append(alerts, policyAndViolationsToAlert(p, violations))
		}
		return nil
	})
	return alerts, nil
}

func policyAndViolationsToAlert(policy *v1.Policy, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: v1.LifecycleStage_BUILD_TIME,
		Policy:         proto.Clone(policy).(*v1.Policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	return alert
}
