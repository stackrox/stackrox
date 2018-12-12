package google

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

const (
	username = "_json_key"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "google", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func validate(google *storage.GoogleConfig) error {
	errorList := errorhelpers.NewErrorList("Google Validation")
	if google.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Google registry (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if google.GetServiceAccount() == "" {
		errorList.AddString("Service account must be specified for Google registry")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration) (*docker.Registry, error) {
	googleConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Google)
	if !ok {
		return nil, fmt.Errorf("Google configuration required")
	}
	config := googleConfig.Google
	if err := validate(config); err != nil {
		return nil, err
	}
	cfg := docker.Config{
		Username: username,
		Password: config.GetServiceAccount(),
		Endpoint: config.GetEndpoint(),
	}
	return docker.NewDockerRegistry(cfg, integration)
}
