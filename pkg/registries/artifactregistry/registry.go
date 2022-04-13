package artifactregistry

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/registries/google"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "artifactregistry", func(integration *storage.ImageIntegration) (types.Registry, error) {
		return google.NewRegistry(integration)
	}
}
