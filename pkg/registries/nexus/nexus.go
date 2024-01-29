package nexus

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.NexusType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := docker.NewRegistryWithoutManifestCall(integration, false)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.NexusType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := docker.NewRegistryWithoutManifestCall(integration, true)
			return reg, err
		}
}
