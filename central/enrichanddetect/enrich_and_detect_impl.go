package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/generated/storage"
)

type enricherAndDetectorImpl struct {
	manager lifecycle.Manager
}

// EnrichAndDetect runs enrichment and detection on a deployment.
func (e *enricherAndDetectorImpl) EnrichAndDetect(deployment *storage.Deployment) error {
	_, _, err := e.manager.DeploymentUpdated(deployment)
	return err
}
