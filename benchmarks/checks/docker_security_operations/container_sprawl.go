package dockersecurityoperations

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type containerSprawlBenchmark struct{}

func (c *containerSprawlBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 6.2",
			Description: "Ensure container sprawl is avoided",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *containerSprawlBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Info(&result)
	utils.AddNotef(&result, "There are '%v' containers in use out of '%v'", len(utils.ContainersRunning), len(utils.ContainersAll))
	return
}

// NewContainerSprawlBenchmark implements CIS-6.2
func NewContainerSprawlBenchmark() utils.Check {
	return &containerSprawlBenchmark{}
}
