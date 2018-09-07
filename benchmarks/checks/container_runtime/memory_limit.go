package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type memoryBenchmark struct{}

func (c *memoryBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.10",
			Description: "Ensure memory usage for container is limited",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *memoryBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.Memory == 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) does not have a memory limit", container.ID, container.Name)
		}
	}
	return
}

// NewMemoryBenchmark implements CIS-5.10
func NewMemoryBenchmark() utils.Check {
	return &memoryBenchmark{}
}
