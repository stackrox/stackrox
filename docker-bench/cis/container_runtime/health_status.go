package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type runtimeHealthcheckBenchmark struct{}

func (c *runtimeHealthcheckBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.26",
			Description: "Ensure container health is checked at runtime",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *runtimeHealthcheckBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.State.Status != "running" {
			continue
		}
		if container.State.Health == nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' does not have health configured", container.ID)
			continue
		}
		if container.State.Health.Status == "" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' is currently reporting empty health", container.ID)
		}
	}
	return
}

// NewRuntimeHealthcheckBenchmark implements CIS-5.26
func NewRuntimeHealthcheckBenchmark() utils.Check {
	return &runtimeHealthcheckBenchmark{}
}
