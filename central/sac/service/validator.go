package service

import (
	"net/url"

	errors "github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
)

func validateConfig(config *storage.AuthzPluginConfig) error {
	if config.GetName() == "" {
		return errors.New("plugin config must specify a name")
	}
	endpoint := config.GetEndpointConfig()
	if endpoint.GetEndpoint() == "" {
		return errors.New("endpoint config must specify an endpoint")
	}
	if _, err := url.Parse(endpoint.GetEndpoint()); err != nil {
		return errors.Wrap(err, "endpoint config must specify a valid URL")
	}
	return nil
}
