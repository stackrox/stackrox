package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type ulimitBenchmark struct{}

func (c *ulimitBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.18",
		Description:  "Ensure the default ulimit is overwritten at runtime, only if needed",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *ulimitBenchmark) Run() (result common.TestResult) {
	result.Note()
	for _, container := range common.ContainersRunning {
		if len(container.HostConfig.Ulimits) > 0 {
			result.AddNotef("Container %v overrides ulimits", container.ID)
		}
	}
	return
}

// NewUlimitBenchmark implements CIS-5.18
func NewUlimitBenchmark() common.Benchmark {
	return &ulimitBenchmark{}
}
