package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type memoryBenchmark struct{}

func (c *memoryBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.10",
			Description: "Ensure memory usage for container is limited",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *memoryBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.Memory == 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v does not have a memory limit", container.ID)
		}
	}
	return
}

// NewMemoryBenchmark implements CIS-5.10
func NewMemoryBenchmark() utils.Check {
	return &memoryBenchmark{}
}
