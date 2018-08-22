package runtime

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Detector provides an interface for performing runtime policy violation detection.
type Detector interface {
	Detect(container *v1.Container) ([]*v1.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
