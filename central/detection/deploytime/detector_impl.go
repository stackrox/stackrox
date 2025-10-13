package deploytime

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
)

type detectorImpl struct {
	policySet      detection.PolicySet
	singleDetector deploytime.Detector
}

// PolicySet retrieves the policy set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on a deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx deploytime.DetectionContext, enhancedDeployment booleanpolicy.EnhancedDeployment, policyFilters ...detection.FilterOption) ([]*storage.Alert, error) {
	alerts, err := d.singleDetector.Detect(ctx, enhancedDeployment, policyFilters...)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}
