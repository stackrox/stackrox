package artifactory

import (
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var log = logging.LoggerForModule()

// Creator provides the type and registries.Creator to add to the registry of image registries.
func Creator() (string, types.Creator) {
	return types.ArtifactoryType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := docker.NewDockerRegistry(integration, false, cfg.GetMetricsHandler())
			if err != nil {
				return nil, err
			}
			return &artifactoryRegistry{Registry: reg}, nil
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.ArtifactoryType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := docker.NewDockerRegistry(integration, true, cfg.GetMetricsHandler())
			if err != nil {
				return nil, err
			}
			return &artifactoryRegistry{Registry: reg}, nil
		}
}

var _ types.Registry = (*artifactoryRegistry)(nil)

type artifactoryRegistry struct {
	*docker.Registry
}

// Test overrides the default docker Test because Artifactory's /v2 ping
// endpoint does not require authentication, so Ping always succeeds even
// with invalid credentials.
func (a *artifactoryRegistry) Test() error {
	_, err := a.Client.Repositories()
	if err != nil {
		log.Errorf("error testing Artifactory integration: %v", err)
		if e, _ := err.(*registry.ClientError); e != nil {
			return errors.Errorf("error testing Artifactory integration (code: %d). Please check Central logs for full error", e.Code())
		}
		return err
	}
	return nil
}
