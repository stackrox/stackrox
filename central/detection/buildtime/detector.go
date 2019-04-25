package buildtime

import (
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/generated/storage"
)

// Detector provides an interface for running build time policy violations.
type Detector interface {
	Detect(image *storage.Image) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
