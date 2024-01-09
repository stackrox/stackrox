package artifactory

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registry of image registries.
func Creator() (string,
	func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error),
) {
	return "artifactory",
		func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error) {
			return docker.NewDockerRegistry(integration, false)
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string,
	func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error),
) {
	return "artifactory",
		func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error) {
			return docker.NewDockerRegistry(integration, true)
		}
}
