package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type restartPolicyBenchmark struct{}

func (c *restartPolicyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.14",
			Description: "Ensure 'on-failure' container restart policy is set to '5'",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *restartPolicyBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.RestartPolicy.Name != "on-failure" || container.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has restart policy '%v' with max retries '%v'", container.ID, container.Name,
				container.HostConfig.RestartPolicy.Name,
				container.HostConfig.RestartPolicy.MaximumRetryCount,
			)
		}
	}
	return
}

// NewRestartPolicyBenchmark implements CIS-5.14
func NewRestartPolicyBenchmark() utils.Check {
	return &restartPolicyBenchmark{}
}
