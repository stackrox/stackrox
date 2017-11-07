package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type restartPolicyBenchmark struct{}

func (c *restartPolicyBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.14",
		Description:  "Ensure 'on-failure' container restart policy is set to '5'",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *restartPolicyBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.RestartPolicy.Name != "on-failure" || container.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
			result.Warn()
			result.AddNotef("Container %v has restart policy %v with max retires %v", container.ID,
				container.HostConfig.RestartPolicy.Name,
				container.HostConfig.RestartPolicy.MaximumRetryCount,
			)
		}
	}
	return
}

// NewRestartPolicyBenchmark implements CIS-5.14
func NewRestartPolicyBenchmark() common.Benchmark {
	return &restartPolicyBenchmark{}
}
