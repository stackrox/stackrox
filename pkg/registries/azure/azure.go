package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var log = logging.LoggerForModule()

var _ types.Registry = (*acr)(nil)

// acr implements docker registry access to Azure container registry. The docker credentials
// are derived from short-lived access tokens. The access token is refreshed as part of the transport.
type acr struct {
	types.Registry

	integration *storage.ImageIntegration
	transport   *azureTransport
}

// Config returns an up to date docker registry configuration.
func (e *acr) Config(ctx context.Context) *types.Config {
	// No need for synchronization if there is no transport.
	if e.transport == nil {
		return e.Registry.Config(ctx)
	}
	if err := e.transport.ensureValid(ctx); err != nil {
		log.Errorf("Failed to ensure access token validity for image integration %q: %v", e.transport.name, err)
	}
	return e.Registry.Config(ctx)
}

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.AzureType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := newRegistry(integration, false, cfg.GetMetricsHandler())
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.AzureType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := newRegistry(integration, true, cfg.GetMetricsHandler())
			return reg, err
		}
}

func validate(cfg *storage.AzureConfig) error {
	errorList := errorhelpers.NewErrorList("Azure container registry validation")
	if cfg.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Azure container registry (e.g. <registry>.azurecr.io)")
	}
	if !cfg.GetWifEnabled() && cfg.GetPassword() == "" {
		errorList.AddString("Password must be specified for Azure container registry")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*acr, error) {
	acrCfg := integration.GetAzure()
	if acrCfg == nil {
		return nil, errors.New("ACR configuration required")
	}
	if err := validate(acrCfg); err != nil {
		return nil, err
	}
	dockerConfig := &docker.Config{
		Endpoint:        acrCfg.GetEndpoint(),
		DisableRepoList: disableRepoList,
		MetricsHandler:  metricsHandler,
		RegistryType:    integration.GetType(),
	}
	reg := &acr{
		integration: integration,
	}
	if !acrCfg.GetWifEnabled() {
		dockerConfig.SetCredentials(acrCfg.GetUsername(), acrCfg.GetPassword())
		dockerRegistry, err := docker.NewDockerRegistryWithConfig(dockerConfig, reg.integration)
		if err != nil {
			return nil, errors.Wrap(err, "creating docker registry")
		}
		reg.Registry = dockerRegistry
		return reg, nil
	}
	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Azure default credentials")
	}
	reg.transport = newAzureTransport(integration.GetName(), dockerConfig, creds)
	dockerRegistry, err := docker.NewDockerRegistryWithConfig(dockerConfig, reg.integration, reg.transport)
	if err != nil {
		return nil, errors.Wrap(err, "creating docker registry")
	}
	reg.Registry = dockerRegistry
	return reg, nil
}
