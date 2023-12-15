package quay

import (
	"io"
	"net/http"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	oauthTokenString = "$oauthtoken"

	timeout = 5 * time.Second
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.QuayType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := newRegistry(integration, false)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.QuayType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := newRegistry(integration, true)
			return reg, err
		}
}

var _ types.Registry = (*Quay)(nil)

// Quay is the implementation of the Docker Registry for Quay
type Quay struct {
	*docker.Registry
	config *storage.QuayConfig
}

func validate(quay *storage.QuayConfig, categories []storage.ImageIntegrationCategory) error {
	if quay.GetEndpoint() == "" {
		return errors.New("Quay endpoint must be specified")
	}

	if features.QuayRobotAccounts.Enabled() {
		if len(categories) == 1 && categories[0] == storage.ImageIntegrationCategory_SCANNER {
			// If scanner only, robot credentials doesn't work. So expect either OAuth token or nothing (public registry)
			if quay.GetRegistryRobotCredentials() != nil {
				return errors.New("Quay scanner integration cannot use robot credentials")
			}
		} else if len(categories) == 1 && categories[0] == storage.ImageIntegrationCategory_REGISTRY {
			// If registry only, only one of OAuth token, robot credentials or neither is allowed. Error if both are provided
			if quay.GetRegistryRobotCredentials() != nil && quay.GetOauthToken() != "" {
				return errors.New("Quay registry integration should use robot credentials or OAuth token but not both")
			}
		} else {
			// If both scanner and registry, then ensure that we don't have robot credentials by itself
			// That implies we have to use robot creds for scanner which is not possible.
			// Both being empty is ok as that's a public registry.
			if quay.GetOauthToken() == "" && quay.GetRegistryRobotCredentials() != nil {
				return errors.New("Quay scanner integration cannot use robot credentials")
			}
		}

		// If using robot creds, check that both username and password is provided.
		if quay.GetRegistryRobotCredentials() != nil {
			if quay.GetRegistryRobotCredentials().GetUsername() == "" || quay.GetRegistryRobotCredentials().GetPassword() == "" {
				return errors.New("Both username and password must be provided when using Quay robot credentials")
			}
		}
	}

	// Note that all credentials could be empty because there are public images
	return nil
}

// NewRegistryFromConfig returns a new instantiation of the Quay registry
func NewRegistryFromConfig(config *storage.QuayConfig, integration *storage.ImageIntegration, disableRepoList bool) (types.Registry, error) {
	if err := validate(config, integration.GetCategories()); err != nil {
		return nil, err
	}

	var username, password string
	password = config.GetOauthToken()

	if features.QuayRobotAccounts.Enabled() {
		if config.GetRegistryRobotCredentials() != nil {
			// If robot credentials are provided use it for registry, regardless of whether ImageIntegration is also for scanner.
			// The scanner portion of it can use OAuth token, but the registry object should use proper robot creds.
			username = config.GetRegistryRobotCredentials().GetUsername()
			password = config.GetRegistryRobotCredentials().GetPassword()
		} else if config.GetOauthToken() != "" {
			username = oauthTokenString
		}
	} else {
		if config.GetOauthToken() != "" {
			username = oauthTokenString
		}
	}

	cfg := &docker.Config{
		Username:        username,
		Password:        password,
		Endpoint:        config.GetEndpoint(),
		Insecure:        config.GetInsecure(),
		DisableRepoList: disableRepoList,
	}
	dockerRegistry, err := docker.NewDockerRegistryWithConfig(cfg, integration)
	if err != nil {
		return nil, err
	}
	return &Quay{
		Registry: dockerRegistry,
		config:   config,
	}, nil
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool) (types.Registry, error) {
	quayConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Quay)
	if !ok {
		return nil, errors.New("Quay config must be specified")
	}
	return NewRegistryFromConfig(quayConfig.Quay, integration, disableRepoList)
}

// Test overrides the default docker Test function because the Quay Ping endpoint requires Auth
func (q *Quay) Test() error {
	if q.config.GetOauthToken() != "" {
		return q.Registry.Test()
	}

	url := urlfmt.FormatURL(q.config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	discoveryURL := url + "/api/v1/discovery"
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		log.Errorf("Quay error response: %d", resp.StatusCode)
		return errors.Errorf("received http status code %d from Quay. Check Central logs for full error.", resp.StatusCode)
	}

	defer utils.IgnoreError(resp.Body.Close)
	if !httputil.Is2xxOr3xxStatusCode(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("error reaching quay.io with HTTP code %d", resp.StatusCode)
		}
		log.Errorf("Quay error response: %d %s", resp.StatusCode, string(body))
		return errors.Errorf("received http status code %d from Quay. Check Central logs for full error.", resp.StatusCode)
	}
	return nil
}
