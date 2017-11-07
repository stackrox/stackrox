package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type cpuPriorityBenchmark struct{}

func (c *cpuPriorityBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.11",
		Description:  "Ensure CPU priority is set appropriately on the container",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *cpuPriorityBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.CPUShares == 0 {
			result.Warn()
			result.AddNotef("Container %v does not have cpu shares set", container.ID)
		}
	}
	return
}

// NewCPUPriorityBenchmark implements CIS-5.11
func NewCPUPriorityBenchmark() common.Benchmark {
	return &cpuPriorityBenchmark{}
}
