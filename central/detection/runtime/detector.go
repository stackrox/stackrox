package runtime

import (
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/generated/api/v1"
)

// Detector provides an interface for performing runtime policy violation detection.
type Detector interface {
	Detect(deployment *v1.Deployment) ([]*v1.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet deployment.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
