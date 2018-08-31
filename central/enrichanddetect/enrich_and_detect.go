package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
)

// EnricherAndDetector combines enrichment and detection into a single function call.
//go:generate mockgen -package mocks -destination mocks/enricher_and_detector.go github.com/stackrox/rox/central/enrichanddetect EnricherAndDetector
type EnricherAndDetector interface {
	EnrichAndDetect(deployment *v1.Deployment) error
}

// New returns a new instance of a EnricherAndDetector.
func New(enricher enrichment.Enricher, detector deploytime.Detector) EnricherAndDetector {
	return &enricherAndDetecterImpl{
		enricher: enricher,
		detector: detector,
	}
}
