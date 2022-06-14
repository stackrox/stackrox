package buildtime

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/detection"
)

// Detector provides an interface for running build time policy violations.
type Detector interface {
	Detect(image *storage.Image, policyFilters ...detection.FilterOption) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
