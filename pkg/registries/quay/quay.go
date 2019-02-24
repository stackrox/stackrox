package quay

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/stackrox/rox/generated/storage"
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
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "quay", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
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
		return fmt.Errorf("Quay endpoint must be specified")
	}
	// Note that the oauth token could be empty because there are public images
	return nil
}

// NewRegistryFromConfig returns a new instantiation of the Quay registry
func NewRegistryFromConfig(config *storage.QuayConfig, integration *storage.ImageIntegration) (types.ImageRegistry, error) {
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

func newRegistry(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
	quayConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Quay)
	if !ok {
		return nil, fmt.Errorf("Quay config must be specified")
	}
	return NewRegistryFromConfig(quayConfig.Quay, integration)
}

// Test overrides the default docker Test function because the Quay Ping endpoint requires Auth
func (q *Quay) Test() error {
	if q.config.GetOauthToken() != "" {
		return q.Registry.Test()
	}

	url, err := urlfmt.FormatURL(q.config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return err
	}
	discoveryURL := url + "/api/v1/discovery"
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		defer utils.IgnoreError(resp.Body.Close)
		if err != nil {
			return fmt.Errorf("Error reaching quay.io with HTTP code %d", resp.StatusCode)
		}
		return fmt.Errorf("Error reaching quay.io with HTTP code %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
