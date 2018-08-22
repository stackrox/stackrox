package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/generated/api/v1"
)

type enricherAndDetecterImpl struct {
	enricher enrichment.Enricher
	detector deploytime.Detector
}

// EnrichAndDetect runs enrichment and detection on a deployment.
func (e *enricherAndDetecterImpl) EnrichAndDetect(deployment *v1.Deployment) error {
	updated, err := e.enricher.Enrich(deployment)
	if err != nil {
		return err
	}
	if updated {
		_, _, err := e.detector.DeploymentUpdated(deployment)
		return err
	}
	return nil
}
