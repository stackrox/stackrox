package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type cgroupBenchmark struct{}

func (c *cgroupBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.24",
		Description:  "Ensure cgroup usage is confirmed",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *cgroupBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.CgroupParent != "docker" && container.HostConfig.CgroupParent != "" {
			result.Warn()
			result.AddNotef("Container %v has the cgroup parent set to %v", container.ID, container.HostConfig.CgroupParent)
		}
	}
	return
}

// NewCgroupBenchmark implements CIS-5.24
func NewCgroupBenchmark() common.Benchmark {
	return &cgroupBenchmark{}
}
