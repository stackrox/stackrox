package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type restartPolicyBenchmark struct{}

func (c *restartPolicyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.14",
			Description: "Ensure 'on-failure' container restart policy is set to '5'",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *restartPolicyBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.RestartPolicy.Name != "on-failure" || container.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' has restart policy '%v' with max retries '%v'", container.ID,
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
