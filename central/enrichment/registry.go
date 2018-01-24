package enrichment

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
)

// UpdateRegistry updates image processors map of active registries
func (e *Enricher) UpdateRegistry(registry registries.ImageRegistry) {
	e.registryMutex.Lock()
	defer e.registryMutex.Unlock()
	e.registries[registry.ProtoRegistry().GetId()] = registry
}

// RemoveRegistry removes a registry from image processors map of active registries
func (e *Enricher) RemoveRegistry(id string) {
	e.registryMutex.Lock()
	defer e.registryMutex.Unlock()
	delete(e.registries, id)
}

func (e *Enricher) enrichWithMetadata(image *v1.Image) (updated bool, err error) {
	e.registryMutex.Lock()
	defer e.registryMutex.Unlock()
	for _, registry := range e.registries {
		if updated, err = e.enrichImageWithRegistry(image, registry); err != nil {
			return
		} else if updated {
			return
		}
	}
	return
}

// EnrichWithRegistry enriches a deployment with a specific registry.
func (e *Enricher) EnrichWithRegistry(deployment *v1.Deployment, registry registries.ImageRegistry) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := e.enrichImageWithRegistry(c.GetImage(), registry); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	if updated {
		e.storage.UpdateDeployment(deployment)
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
		e.storage.UpdateImage(image)
		return true, nil
	}

	return false, nil
}
