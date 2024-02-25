package ibm

import (
	"errors"
	"time"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

const (
	username = "iamapikey"

	registryTimeout = 10 * time.Second
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.IBMType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			return newRegistry(integration, false)
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.IBMType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			return newRegistry(integration, true)
		}
}

func validate(ibm *storage.IBMRegistryConfig) error {
	var validationErrs error
	if ibm.GetEndpoint() == "" {
		validationErrs = errors.Join(validationErrs,
			errors.New("endpoint must be specified for IBM registry (e.g. us.icr.io)"))
	}
	if ibm.GetApiKey() == "" {
		validationErrs = errors.Join(validationErrs,
			errors.New("IAM API Key must be specified for IBM registry"))
	}
	return pkgErrors.Wrap(validationErrs, "validating config")
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool) (*docker.Registry, error) {
	ibmConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Ibm)
	if !ok {
		return nil, errors.New("IBM configuration required")
	}
	config := ibmConfig.Ibm
	if err := validate(config); err != nil {
		return nil, err
	}
	cfg := &docker.Config{
		Username:        username,
		Password:        config.GetApiKey(),
		Endpoint:        config.GetEndpoint(),
		DisableRepoList: disableRepoList,
	}
	registry, err := docker.NewDockerRegistryWithConfig(cfg, integration)
	if err != nil {
		return nil, err
	}
	// IBM needs a custom timeout because it's pretty slow
	registry.Client.Client.Timeout = registryTimeout
	return registry, nil
}
