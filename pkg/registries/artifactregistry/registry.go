package artifactregistry

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/google"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "artifactregistry", func(integration *storage.ImageIntegration) (types.Registry, error) {
		return google.NewRegistry(integration, false)
	}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "artifactregistry", func(integration *storage.ImageIntegration) (types.Registry, error) {
		return google.NewRegistry(integration, true)
	}
}
