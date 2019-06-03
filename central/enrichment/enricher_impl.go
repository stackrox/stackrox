package enrichment

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/types"
)

// enricherImpl enriches images with data from registries and scanners.
type enricherImpl struct {
	imageEnricher enricher.ImageEnricher
}

// EnrichDeployment enriches a deployment with data from registries and scanners.
func (e *enricherImpl) EnrichDeployment(ctx enricher.EnrichmentContext, deployment *storage.Deployment) ([]*storage.Image, []int, error) {
	return e.enrichDeployment(ctx, deployment)
}

func (e *enricherImpl) enrichDeployment(ctx enricher.EnrichmentContext, deployment *storage.Deployment) ([]*storage.Image, []int, error) {
	var (
		images         []*storage.Image
		updatedIndices []int
	)
	for i, c := range deployment.GetContainers() {
		img := types.ToImage(c.GetImage())
		images = append(images, img)
		if updated := e.imageEnricher.EnrichImage(ctx, img); updated && img.GetId() != "" {
			updatedIndices = append(updatedIndices, i)
		}
	}
	return images, updatedIndices, nil
}
