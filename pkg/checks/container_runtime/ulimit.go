package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type ulimitBenchmark struct{}

func (c *ulimitBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.18",
			Description: "Ensure the default ulimit is overwritten at runtime, only if needed",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *ulimitBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.Ulimits) > 0 {
			utils.AddNotef(&result, "Container '%v' overrides ulimits", container.ID)
		}
	}
	return
}

// NewUlimitBenchmark implements CIS-5.18
func NewUlimitBenchmark() utils.Check {
	return &ulimitBenchmark{}
}
