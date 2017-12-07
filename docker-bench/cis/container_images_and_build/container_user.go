package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type containerUserBenchmark struct{}

func (c *containerUserBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.1",
			Description: "Ensure a user for the container has been created",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *containerUserBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.Config.User == "" || container.Config.User == "root" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v with image %v is running as root user", container.ID, container.Config.Image)
		}
	}
	return
}

// NewContainerUserBenchmark implements CIS-4.1
func NewContainerUserBenchmark() utils.Check {
	return &containerUserBenchmark{}
}
