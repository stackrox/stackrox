package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type runtimeHealthcheckBenchmark struct{}

func (c *runtimeHealthcheckBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.26",
		Description:  "Ensure container health is checked at runtime",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *runtimeHealthcheckBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.State.Status != "running" {
			continue
		}
		if container.State.Health == nil {
			result.Warn()
			result.AddNotef("Container %v does not have health configured", container.ID)
			continue
		}
		if container.State.Health.Status == "" {
			result.Warn()
			result.AddNotef("Container %v is currently reporting empty health", container.ID)
		}
	}
	return
}

// NewRuntimeHealthcheckBenchmark implements CIS-5.26
func NewRuntimeHealthcheckBenchmark() common.Benchmark {
	return &runtimeHealthcheckBenchmark{}
}
