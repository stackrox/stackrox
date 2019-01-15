package enrichanddetect

import (
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
)

type enricherAndDetectorImpl struct {
	enricher            enrichment.Enricher
	manager             lifecycle.Manager
	deploymentDatastore deploymentDatastore.DataStore
	imageDatastore      imageDatastore.DataStore
}

// EnrichAndDetect runs enrichment and detection on a deployment.
func (e *enricherAndDetectorImpl) EnrichAndDetect(deployment *storage.Deployment) error {
	updatedImages, updated, err := e.enricher.EnrichDeployment(deployment)
	if err != nil {
		log.Errorf("Error enriching deployment %s: %s", deployment.GetName(), err)
		return nil
	}
	if updated {
		for _, i := range updatedImages {
			if err := e.imageDatastore.UpsertImage(i); err != nil {
				log.Errorf("Error persisting image %s: %s", i.GetName().GetFullName(), err)
			}
		}
		if err := e.deploymentDatastore.UpdateDeployment(deployment); err != nil {
			log.Errorf("Error persisting deployment %s: %s", deployment.GetName(), err)
		}
	}
	return nil
}
