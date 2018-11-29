package dockerdaemonconfiguration

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type aufsBenchmark struct{}

func (c *aufsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.5",
			Description: "Ensure aufs storage driver is not used",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *aufsBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if strings.Contains(utils.DockerInfo.Driver, "aufs") {
		utils.Warn(&result)
		utils.AddNotes(&result, "aufs is currently configured as the storage driver")
		return
	}
	utils.Pass(&result)
	return
}

// NewAUFSBenchmark implements CIS-2.5
func NewAUFSBenchmark() utils.Check {
	return &aufsBenchmark{}
}
