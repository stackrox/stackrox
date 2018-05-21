package enrichment

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
)

func (e *Enricher) enrichWithMetadata(image *v1.Image) (updated bool, err error) {
	for _, integration := range e.imageIntegrations {
		if integration.Registry == nil {
			continue
		}
		if updated, err = e.enrichImageWithRegistry(image, integration.Registry); err != nil {
			logger.Errorf("Error enriching with registry %s", integration.Name)
			continue
		} else if updated {
			return
		}
	}
	return
}

// enrichWithRegistry enriches a deployment with a specific registry.
func (e *Enricher) enrichWithRegistry(deployment *v1.Deployment, registry registries.ImageRegistry) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := e.enrichImageWithRegistry(c.GetImage(), registry); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	if updated {
		if err := e.deploymentStorage.UpdateDeployment(deployment); err != nil {
			logger.Errorf("unable to update deployment: %s", err)
		}
	}

	return
}

func (e *Enricher) enrichImageWithRegistry(image *v1.Image, registry registries.ImageRegistry) (bool, error) {
	if !registry.Global() {
		return false, nil
	}
	if !registry.Match(image) {
		return false, nil
	}
	// Wait until limiter allows entrance
	var metadata *v1.ImageMetadata
	metadataItem := e.metadataCache.Get(image.GetName().GetFullName())
	if metadataItem == nil {
		metrics.IncrementMetadataCacheMiss()
		e.metadataLimiter.Wait(context.Background())

		var err error
		metadata, err = registry.Metadata(image)
		if err != nil {
			logger.Error(err)
			return false, err
		}
		e.metadataCache.Set(image.GetName().GetFullName(), metadata, imageDataExpiration)
	} else {
		metrics.IncrementMetadataCacheHit()
		metadata = metadataItem.Value().(*v1.ImageMetadata)
	}

	if protoconv.CompareProtoTimestamps(image.GetMetadata().GetCreated(), metadata.GetCreated()) != 0 {
		image.Metadata = metadata
		if err := e.imageStorage.UpdateImage(image); err != nil {
			logger.Errorf("unable to update image: %s", err)
			return false, nil
		}
		return true, nil
	}

	return false, nil
}
