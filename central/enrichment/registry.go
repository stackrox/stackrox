package enrichment

import (
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
			return
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
		if err := e.storage.UpdateDeployment(deployment); err != nil {
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
	metadata, err := registry.Metadata(image)
	if err != nil {
		logger.Error(err)
		return false, err
	}

	if protoconv.CompareProtoTimestamps(image.GetMetadata().GetCreated(), metadata.GetCreated()) != 0 {
		image.Metadata = metadata
		if err := e.storage.UpdateImage(image); err != nil {
			logger.Errorf("unable to update image: %s", err)
			return false, nil
		}
		return true, nil
	}

	return false, nil
}
