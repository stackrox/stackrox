package deploytime

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/logging"
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
func (d *detectorImpl) Detect(ctx deploytime.DetectionContext, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Alert, error) {
	return d.singleDetector.Detect(ctx, deployment, images)
}
