package enrichment

import (
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
)

// enricherImpl enriches images with data from registries and scanners.
type enricherImpl struct {
	imageEnricher enricher.ImageEnricher
	scorer        risk.Scorer
}

// EnrichDeployment enriches a deployment with data from registries and scanners.
func (e *enricherImpl) EnrichDeployment(deployment *storage.Deployment) ([]*storage.Image, bool, error) {
	return e.enrichDeployment(deployment, true)
}

// EnrichDeployment enriches a deployment with data from registries and scanners.
func (e *enricherImpl) EnrichDeploymentWithEmptyImages(deployment *storage.Deployment) ([]*storage.Image, bool, error) {
	return e.enrichDeployment(deployment, false)
}

func (e *enricherImpl) enrichDeployment(deployment *storage.Deployment, ignoreEmptyImages bool) ([]*storage.Image, bool, error) {
	var updatedImages []*storage.Image
	for _, c := range deployment.GetContainers() {
		if c.GetImage().GetId() == "" && ignoreEmptyImages {
			continue
		}
		if updated := e.imageEnricher.EnrichImage(c.Image); updated && c.GetImage().GetId() != "" {
			updatedImages = append(updatedImages, c.GetImage())
		}
	}
	return updatedImages, len(updatedImages) != 0, nil
}
