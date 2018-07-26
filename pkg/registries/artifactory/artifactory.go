package artifactory

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries/docker"
	"bitbucket.org/stack-rox/apollo/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *v1.ImageIntegration) (types.ImageRegistry, error)) {
	_, dockerCreator := docker.Creator()
	return "artifactory", dockerCreator
}
