package buildtime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// PolicySet retrieves the policy set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on an image, returning any generated alerts.  If policy categories are specified, we will only
// run policies with the specified categories
func (d *detectorImpl) Detect(image *storage.Image, policyFilters ...detection.FilterOption) ([]*storage.Alert, error) {
	if image == nil {
		return nil, errors.New("cannot detect on a nil image")
	}

	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		for _, filter := range policyFilters {
			if !filter(compiled.Policy()) {
				return nil
			}
		}
		if !compiled.AppliesTo(image) {
			return nil
		}
		violations, err := compiled.MatchAgainstImage(&cacheReceptacle, image)
		if err != nil {
			return errors.Wrapf(err, "matching against policy %s", compiled.Policy().GetName())
		}
		alertViolations := violations.AlertViolations
		if len(alertViolations) > 0 {
			alerts = append(alerts, policyViolationsAndImageToAlert(compiled.Policy(), alertViolations, image))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func policyViolationsAndImageToAlert(policy *storage.Policy, violations []*storage.Alert_Violation, image *storage.Image) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &storage.Alert{}
	alert.SetId(uuid.NewV4().String())
	alert.SetLifecycleStage(storage.LifecycleStage_BUILD)
	alert.SetPolicy(policy.CloneVT())
	alert.SetImage(proto.ValueOrDefault(types.ToContainerImage(image)))
	alert.SetViolations(violations)
	alert.SetTime(protocompat.TimestampNow())
	return alert
}
