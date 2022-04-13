package rhel

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/registries/docker"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

const (
	// RedHatRegistryType exports the type of the Red Hat registry integration
	RedHatRegistryType = "rhel"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return RedHatRegistryType, func(integration *storage.ImageIntegration) (types.Registry, error) {
		reg, err := docker.NewRegistryWithoutManifestCall(integration)
		return reg, err
	}
}
