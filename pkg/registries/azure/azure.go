package azure

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
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
	errorList := errorhelpers.NewErrorList("azure container registry validation")
	if cfg.GetEndpoint() == "" {
		errorList.AddString("endpoint must be specified for Azure container registry (e.g. <registry>.azurecr.io)")
	}
	if !cfg.GetWifEnabled() && cfg.GetPassword() == "" {
		errorList.AddString("password or workload identity must be specified for Azure container registry")
	}
	return errorList.ToError()
}

func getACRConfig(integration *storage.ImageIntegration) (*storage.AzureConfig, error) {
	// Note that integrations of type "azure" support both `DockerConfig` (deprecated in 4.7) and `AzureConfig`.
	// If the `DockerConfig` schema is used, we convert to `AzureConfig`.
	// TODO(ROX-27720): remove support for `DockerConfig`.
	acrCfg := integration.GetAzure()
	if acrCfg == nil {
		dockerCfg := integration.GetDocker()
		if dockerCfg == nil {
			return nil, errors.New("azure container registry or docker configuration required")
		}
		acrCfg = &storage.AzureConfig{
			Endpoint: dockerCfg.GetEndpoint(),
			Username: dockerCfg.GetUsername(),
			Password: dockerCfg.GetPassword(),
		}
	}
	if err := validate(acrCfg); err != nil {
		return nil, err
	}
	return acrCfg, nil
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*acr, error) {
	acrCfg, err := getACRConfig(integration)
	if err != nil {
		return nil, errors.Wrap(err, "getting Azure container registry configuration")
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
		// Fall back to docker registry with hardcoded credentials.
		dockerConfig.SetCredentials(acrCfg.GetUsername(), acrCfg.GetPassword())
		dockerRegistry, err := docker.NewDockerRegistryWithConfig(dockerConfig, reg.integration)
		if err != nil {
			return nil, errors.Wrap(err, "creating docker registry")
		}
		reg.Registry = dockerRegistry
		return reg, nil
	}

	// Read the credentials from the environment via the Azure default chain.
	credOpts := &azidentity.DefaultAzureCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{Transport: proxy.RoundTripper()},
		},
	}
	creds, err := azidentity.NewDefaultAzureCredential(credOpts)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Azure default credentials")
	}
	authOpts := &azcontainerregistry.AuthenticationClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{Transport: proxy.RoundTripper()},
		},
	}
	// The Azure SDK expects a valid https scheme without slash.
	authClient, err := azcontainerregistry.NewAuthenticationClient(
		urlfmt.FormatURL(dockerConfig.Endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash), authOpts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "creating Azure container registry authentication client")
	}

	reg.transport = newAzureTransport(integration.GetName(), dockerConfig, creds, authClient)
	dockerRegistry, err := docker.NewDockerRegistryWithConfig(dockerConfig, reg.integration, reg.transport)
	if err != nil {
		return nil, errors.Wrap(err, "creating docker registry")
	}
	reg.Registry = dockerRegistry
	return reg, nil
}
