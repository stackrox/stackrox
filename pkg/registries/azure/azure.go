package azure

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.AzureType,
		func(integration *storage.ImageIntegration, metricsHandler *types.MetricsHandler, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := docker.NewDockerRegistry(integration, false, metricsHandler)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.AzureType,
		func(integration *storage.ImageIntegration, metricsHandler *types.MetricsHandler, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := docker.NewDockerRegistry(integration, true, metricsHandler)
			return reg, err
		}
}
