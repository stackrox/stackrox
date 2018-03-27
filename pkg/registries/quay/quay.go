package quay

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/registries/docker"
)

const (
	username = "$oauthtoken"
)

func newRegistry(integration *v1.ImageIntegration) (*docker.Registry, error) {
	cfg := docker.Config{
		Username: username,
		Password: integration.Config["oauthToken"],
		Endpoint: integration.Config["endpoint"],
	}
	return docker.NewDockerRegistry(cfg, integration)
}

func init() {
	registries.Registry["quay"] = func(integration *v1.ImageIntegration) (registries.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}
