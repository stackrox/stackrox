package dockersecurityoperations

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type containerSprawlBenchmark struct{}

func (c *containerSprawlBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 6.2",
		Description:  "Ensure container sprawl is avoided",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *containerSprawlBenchmark) Run() (result common.TestResult) {
	result.Info()
	result.AddNotef("There are %v containers in use out of %v", len(common.ContainersRunning), len(common.ContainersAll))
	return
}

// NewContainerSprawlBenchmark implements CIS-6.2
func NewContainerSprawlBenchmark() common.Benchmark {
	return &containerSprawlBenchmark{}
}
