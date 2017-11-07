package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type memoryBenchmark struct{}

func (c *memoryBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.10",
		Description:  "Ensure memory usage for container is limited",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *memoryBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.Memory == 0 {
			result.Warn()
			result.AddNotef("Container %v does not have a memory limit", container.ID)
		}
	}
	return
}

// NewMemoryBenchmark implements CIS-5.10
func NewMemoryBenchmark() common.Benchmark {
	return &memoryBenchmark{}
}
