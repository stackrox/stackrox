package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type cgroupUsageBenchmark struct{}

func (c *cgroupUsageBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.9",
		Description:  "Ensure the default cgroup usage has been confirmed",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *cgroupUsageBenchmark) Run() (result common.TestResult) {
	if parent, ok := common.DockerConfig["cgroup-parent"]; ok {
		result.Warn()
		result.AddNotef("Cgroup path is set as %v", parent)
		return
	}
	result.Pass()
	return
}

// NewCgroupUsageBenchmark implements CIS-2.9
func NewCgroupUsageBenchmark() common.Benchmark {
	return &cgroupUsageBenchmark{}
}
