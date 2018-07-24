package imageenricher

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

// IntegrationSet provides an interface for interaction with a group of image integrations.
type IntegrationSet interface {
	UpdateImageIntegration(integration *sources.ImageIntegration)
	RemoveImageIntegration(id string)

	GetRegistryMetadataByImage(image *v1.Image) *registries.Config
	Match(image *v1.Image) bool

	GetAll() []*sources.ImageIntegration
}

// NewIntegrationSet returns a new IntegrationSet instance.
func NewIntegrationSet() IntegrationSet {
	return &integrationSetImpl{
		integrations: make(map[string]*sources.ImageIntegration),
	}
}
