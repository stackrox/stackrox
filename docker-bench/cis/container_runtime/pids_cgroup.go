package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type pidCgroupBenchmark struct{}

func (c *pidCgroupBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.28",
		Description:  "Ensure PIDs cgroup limit is used",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *pidCgroupBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.PidsLimit <= 0 {
			result.Warn()
			result.AddNotef("Container %v does not have pids limit set", container.ID)
		}
	}
	return
}

// NewPidCgroupBenchmark implements CIS-5.28
func NewPidCgroupBenchmark() common.Benchmark {
	return &pidCgroupBenchmark{}
}
