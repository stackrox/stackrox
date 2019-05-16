package tenable

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var (
	remote = "registry.cloud.tenable.com"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "tenable", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func validate(config *storage.TenableConfig) error {
	errorList := errorhelpers.NewErrorList("Tenable Validation")
	if config.GetAccessKey() == "" {
		errorList.AddString("Access key must be specified for Tenable scanner")
	}
	if config.GetSecretKey() == "" {
		errorList.AddString("Secret Key must be specified for Tenable scanner")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
	tenableConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Tenable)
	if !ok {
		return nil, fmt.Errorf("tenable configuration required")
	}
	config := tenableConfig.Tenable
	if err := validate(config); err != nil {
		return nil, err
	}

	cfg := docker.Config{
		Endpoint: remote,
		Username: config.GetAccessKey(),
		Password: config.GetSecretKey(),
	}
	return docker.NewDockerRegistryWithConfig(cfg, integration)
}
