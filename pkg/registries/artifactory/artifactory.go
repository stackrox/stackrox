package artifactory

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/registries/docker"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

var (
	logger = logging.LoggerForModule()
)

// Creator provides the type and registries.Creator to add to the registry of image registries.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "artifactory", newRegistry
}

type registry struct {
	*docker.Registry
}

func newRegistry(integration *storage.ImageIntegration) (types.Registry, error) {
	dockerRegistry, err := docker.NewDockerRegistry(integration)
	if err != nil {
		return nil, err
	}
	return &registry{
		Registry: dockerRegistry,
	}, nil
}

// Test implements a valid Test function for Artifactory
func (r *registry) Test() error {
	_, err := r.Client.Repositories()
	if err != nil {
		logger.Errorf("error testing Artifactory integration: %v", err)
	}
	return err
}
