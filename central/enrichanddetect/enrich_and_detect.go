package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/generated/storage"
)

// EnricherAndDetector combines enrichment and detection into a single function call.
//go:generate mockgen-wrapper EnricherAndDetector
type EnricherAndDetector interface {
	EnrichAndDetect(deployment *storage.Deployment) error
}

// New returns a new instance of a EnricherAndDetector.
func New(manager lifecycle.Manager) EnricherAndDetector {
	return &enricherAndDetectorImpl{
		manager: manager,
	}
}
