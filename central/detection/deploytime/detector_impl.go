package deploytime

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy"
	"github.com/stackrox/stackrox/pkg/detection"
	"github.com/stackrox/stackrox/pkg/detection/deploytime"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type detectorImpl struct {
	policySet      detection.PolicySet
	singleDetector deploytime.Detector
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(ctx deploytime.DetectionContext, enhancedDeployment booleanpolicy.EnhancedDeployment, policyFilters ...detection.FilterOption) ([]*storage.Alert, error) {
	alerts, err := d.singleDetector.Detect(ctx, enhancedDeployment, policyFilters...)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}
