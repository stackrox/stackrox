package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
)

type enricherAndDetectorImpl struct {
	manager lifecycle.Manager
}

// EnrichAndDetect runs enrichment and detection on a deployment.
func (e *enricherAndDetectorImpl) EnrichAndDetect(deployment *storage.Deployment) error {
	return e.manager.DeploymentUpdated(enricher.EnrichmentContext{IgnoreExisting: true}, deployment, nil)
}
