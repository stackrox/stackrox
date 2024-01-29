package artifactregistry

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/google"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.ArtifactRegistryType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return google.NewRegistry(integration, false, cfg.GetGCPTokenManager())
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.ArtifactRegistryType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return google.NewRegistry(integration, true, cfg.GetGCPTokenManager())
		}
}
