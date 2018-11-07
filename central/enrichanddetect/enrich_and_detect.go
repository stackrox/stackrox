package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
)

// EnricherAndDetector combines enrichment and detection into a single function call.
//go:generate mockgen-wrapper EnricherAndDetector
type EnricherAndDetector interface {
	EnrichAndDetect(deployment *v1.Deployment) error
}

// New returns a new instance of a EnricherAndDetector.
func New(enricher enrichment.Enricher, manager lifecycle.Manager) EnricherAndDetector {
	return &enricherAndDetectorImpl{
		enricher: enricher,
		manager:  manager,
	}
}
