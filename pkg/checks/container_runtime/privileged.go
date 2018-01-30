package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type privilegedBenchmark struct{}

func (c *privilegedBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.4",
			Description: "Ensure privileged containers are not used",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *privilegedBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.Privileged {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' is running as privileged", container.ID)
		}
	}
	return
}

// NewPrivilegedBenchmark implements CIS-5.4
func NewPrivilegedBenchmark() utils.Check {
	return &privilegedBenchmark{}
}
