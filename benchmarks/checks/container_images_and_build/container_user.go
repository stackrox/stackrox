package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type containerUserBenchmark struct{}

func (c *containerUserBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.1",
			Description: "Ensure a user for the container has been created",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *containerUserBenchmark) Run() (result storage.BenchmarkCheckResult) {
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
