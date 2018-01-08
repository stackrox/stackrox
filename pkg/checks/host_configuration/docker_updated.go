package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
)

type dockerUpdated struct{}

func (c *dockerUpdated) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 1.3",
			Description: "Ensure Docker is up to date",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *dockerUpdated) Run() (result v1.CheckResult) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	version, err := utils.DockerClient.ServerVersion(ctx)
	if err != nil {
		utils.Note(&result)
		utils.AddNotef(&result, "Manual introspection will be req'd for docker version. Could not retrieve due to %+v", err)
		return
	}
	utils.Note(&result)
	utils.AddNotef(&result, "Docker server is currently running '%v'", version.Version)
	return
}

// NewDockerUpdated implements CIS-1.3
func NewDockerUpdated() utils.Check {
	return &dockerUpdated{}
}
