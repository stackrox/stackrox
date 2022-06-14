package ibm

import (
	"errors"
	"time"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/registries/docker"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

const (
	username = "iamapikey"

	registryTimeout = 10 * time.Second
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "ibm", func(integration *storage.ImageIntegration) (types.Registry, error) {
		return newRegistry(integration)
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

func newRegistry(integration *storage.ImageIntegration) (*docker.Registry, error) {
	ibmConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Ibm)
	if !ok {
		return nil, errors.New("IBM configuration required")
	}
	config := ibmConfig.Ibm
	if err := validate(config); err != nil {
		return nil, err
	}
	cfg := docker.Config{
		Username: username,
		Password: config.GetApiKey(),
		Endpoint: config.GetEndpoint(),
	}
	registry, err := docker.NewDockerRegistryWithConfig(cfg, integration)
	if err != nil {
		return nil, err
	}
	// IBM needs a custom timeout because it's pretty slow
	registry.Client.Client.Timeout = registryTimeout
	return registry, nil
}
