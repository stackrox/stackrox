package azure

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "azure", func(integration *storage.ImageIntegration) (types.Registry, error) {
		reg, err := docker.NewDockerRegistry(integration)
		return reg, err
	}
}
