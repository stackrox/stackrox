package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type pidNamespaceBenchmark struct{}

func (c *pidNamespaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.15",
			Description: "Ensure the host's process namespace is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *pidNamespaceBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.PidMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' has pid mode set to host", container.ID)
		}
	}
	return
}

// NewPidNamespaceBenchmark implements CIS-5.15
func NewPidNamespaceBenchmark() utils.Check {
	return &pidNamespaceBenchmark{}
}
