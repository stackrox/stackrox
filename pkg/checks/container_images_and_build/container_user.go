package containerimagesandbuild

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type containerUserBenchmark struct{}

func (c *containerUserBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.1",
			Description: "Ensure a user for the container has been created",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *containerUserBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.Config.User == "" || container.Config.User == "root" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' with image '%v' is running as root user", container.ID, container.Config.Image)
		}
	}
	return
}

// NewContainerUserBenchmark implements CIS-4.1
func NewContainerUserBenchmark() utils.Check {
	return &containerUserBenchmark{}
}
