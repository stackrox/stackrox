package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type cpuPriorityBenchmark struct{}

func (c *cpuPriorityBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.11",
			Description: "Ensure CPU priority is set appropriately on the container",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *cpuPriorityBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.CPUShares == 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' does not have cpu shares set", container.ID)
		}
	}
	return
}

// NewCPUPriorityBenchmark implements CIS-5.11
func NewCPUPriorityBenchmark() utils.Check {
	return &cpuPriorityBenchmark{}
}
