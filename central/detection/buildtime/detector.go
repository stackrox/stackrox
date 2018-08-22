package buildtime

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Detector provides an interface for running build time policy violations.
type Detector interface {
	Detect(image *v1.Image) ([]*v1.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
