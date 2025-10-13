package buildtime

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
)

// Detector provides an interface for running build time policy violations.
type Detector interface {
	PolicySet() detection.PolicySet

	Detect(image *storage.Image, policyFilters ...detection.FilterOption) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
