package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type ulimitBenchmark struct{}

func (c *ulimitBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.18",
			Description: "Ensure the default ulimit is overwritten at runtime, only if needed",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *ulimitBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.Ulimits) > 0 {
			utils.AddNotef(&result, "Container '%v' (%v) overrides ulimits", container.ID, container.Name)
		}
	}
	return
}

// NewUlimitBenchmark implements CIS-5.18
func NewUlimitBenchmark() utils.Check {
	return &ulimitBenchmark{}
}
