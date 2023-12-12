package rhel

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// RedHatRegistryType exports the type of the Red Hat registry integration
	RedHatRegistryType = "rhel"
)

// RedHatRegistryEndpoints represents endpoints for RHEL registries that should
// use this registry implementation (Metadata invocations may fail otherwise)
var RedHatRegistryEndpoints = set.NewFrozenSet("registry.redhat.io")

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string,
	func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error),
) {
	return RedHatRegistryType,
		func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error) {
			reg, err := docker.NewRegistryWithoutManifestCall(integration, false)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string,
	func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error),
) {
	return RedHatRegistryType,
		func(integration *storage.ImageIntegration, _ *types.CreatorOptions) (types.Registry, error) {
			reg, err := docker.NewRegistryWithoutManifestCall(integration, true)
			return reg, err
		}
}
