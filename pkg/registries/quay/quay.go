package quay

import (
	"io"
	"net/http"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/registries/docker"
	"github.com/stackrox/stackrox/pkg/registries/types"
	"github.com/stackrox/stackrox/pkg/urlfmt"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	oauthTokenString = "$oauthtoken"

	timeout = 5 * time.Second
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "quay", func(integration *storage.ImageIntegration) (types.Registry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

// Quay is the implementation of the Docker Registry for Quay
type Quay struct {
	*docker.Registry
	config *storage.QuayConfig
}

func validate(quay *storage.QuayConfig) error {
	if quay.GetEndpoint() == "" {
		return errors.New("Quay endpoint must be specified")
	}
	// Note that the oauth token could be empty because there are public images
	return nil
}

// NewRegistryFromConfig returns a new instantiation of the Quay registry
func NewRegistryFromConfig(config *storage.QuayConfig, integration *storage.ImageIntegration) (types.Registry, error) {
	if err := validate(config); err != nil {
		return nil, err
	}

	var username string
	if config.GetOauthToken() != "" {
		username = oauthTokenString
	}

	cfg := docker.Config{
		Username: username,
		Password: config.GetOauthToken(),
		Endpoint: config.GetEndpoint(),
		Insecure: config.GetInsecure(),
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

func newRegistry(integration *storage.ImageIntegration) (types.Registry, error) {
	quayConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Quay)
	if !ok {
		return nil, errors.New("Quay config must be specified")
	}
	return NewRegistryFromConfig(quayConfig.Quay, integration)
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
