package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type restartPolicyBenchmark struct{}

func (c *restartPolicyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.14",
			Description: "Ensure 'on-failure' container restart policy is set to '5'",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *restartPolicyBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.RestartPolicy.Name != "on-failure" || container.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v has restart policy %v with max retires %v", container.ID,
				container.HostConfig.RestartPolicy.Name,
				container.HostConfig.RestartPolicy.MaximumRetryCount,
			)
		}
	}
	return
}

// NewRestartPolicyBenchmark implements CIS-5.14
func NewRestartPolicyBenchmark() utils.Benchmark {
	return &restartPolicyBenchmark{}
}
