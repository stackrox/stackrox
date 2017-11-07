package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type privilegedBenchmark struct{}

func (c *privilegedBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.4",
		Description:  "Ensure privileged containers are not used",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *privilegedBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.Privileged {
			result.Warn()
			result.AddNotef("Container %v is running as privileged", container.ID)
		}
	}
	return
}

// NewPrivilegedBenchmark implements CIS-5.4
func NewPrivilegedBenchmark() common.Benchmark {
	return &privilegedBenchmark{}
}
