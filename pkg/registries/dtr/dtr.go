package dtr

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/registries/docker"
)

func newRegistry(integration *v1.ImageIntegration) (*docker.Registry, error) {
	dtrConfig, ok := integration.IntegrationConfig.(*v1.ImageIntegration_Dtr)
	if !ok {
		return nil, fmt.Errorf("DTR configuration required")
	}
	cfg := docker.Config{
		Username: dtrConfig.Dtr.GetUsername(),
		Password: dtrConfig.Dtr.GetPassword(),
		Endpoint: dtrConfig.Dtr.GetEndpoint(),
		Insecure: dtrConfig.Dtr.GetInsecure(),
	}
	return docker.NewDockerRegistry(cfg, integration)
}

func init() {
	registries.Registry["dtr"] = func(integration *v1.ImageIntegration) (registries.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}
