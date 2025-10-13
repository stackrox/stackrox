package ibm

import (
	"errors"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

const (
	username = "iamapikey"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.IBMType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return newRegistry(integration, false, cfg.GetMetricsHandler())
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.IBMType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return newRegistry(integration, true, cfg.GetMetricsHandler())
		}
}

func validate(ibm *storage.IBMRegistryConfig) error {
	errorList := errorhelpers.NewErrorList("IBM Validation")
	if ibm.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for IBM registry (e.g. us.icr.io)")
	}
	if ibm.GetApiKey() == "" {
		errorList.AddString("IAM API Key must be specified for IBM registry")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*docker.Registry, error) {
	ibmConfig, ok := integration.GetIntegrationConfig().(*storage.ImageIntegration_Ibm)
	if !ok {
		return nil, errors.New("IBM configuration required")
	}
	config := ibmConfig.Ibm
	if err := validate(config); err != nil {
		return nil, err
	}
	cfg := &docker.Config{
		Endpoint:        config.GetEndpoint(),
		DisableRepoList: disableRepoList,
		MetricsHandler:  metricsHandler,
		RegistryType:    integration.GetType(),
	}
	cfg.SetCredentials(username, config.GetApiKey())
	registry, err := docker.NewDockerRegistryWithConfig(cfg, integration)
	if err != nil {
		return nil, err
	}
	return registry, nil
}
