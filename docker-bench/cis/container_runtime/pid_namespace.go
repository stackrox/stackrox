package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type pidNamespaceBenchmark struct{}

func (c *pidNamespaceBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.15",
		Description:  "Ensure the host's process namespace is not shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *pidNamespaceBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.PidMode.IsHost() {
			result.Warn()
			result.AddNotef("Container %v has pid mode set to host", container.ID)
		}
	}
	return
}

// NewPidNamespaceBenchmark implements CIS-5.15
func NewPidNamespaceBenchmark() common.Benchmark {
	return &pidNamespaceBenchmark{}
}
