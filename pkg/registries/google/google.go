package google

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/registries/docker"
)

const (
	username = "_json_key"
)

func newRegistry(integration *v1.ImageIntegration) (*docker.Registry, error) {
	if _, ok := integration.Config["endpoint"]; !ok {
		return nil, fmt.Errorf("Endpoint must be specified for Google registry (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if _, ok := integration.Config["serviceAccount"]; !ok {
		return nil, fmt.Errorf("Service account must be specified for Google registry")
	}
	cfg := docker.Config{
		Username: username,
		Password: integration.Config["serviceAccount"],
		Endpoint: integration.Config["endpoint"],
	}
	return docker.NewDockerRegistry(cfg, integration)
}

func init() {
	registries.Registry["google"] = func(integration *v1.ImageIntegration) (registries.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}
