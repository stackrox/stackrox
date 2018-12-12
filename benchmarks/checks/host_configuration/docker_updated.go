package hostconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
)

type dockerUpdated struct{}

func (c *dockerUpdated) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 1.3",
			Description: "Ensure Docker is up to date",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *dockerUpdated) Run() (result storage.BenchmarkCheckResult) {
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
