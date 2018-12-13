package buildtime

import (
	"github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/generated/storage"
)

// Detector provides an interface for running build time policy violations.
type Detector interface {
	Detect(image *storage.Image) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet image.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
