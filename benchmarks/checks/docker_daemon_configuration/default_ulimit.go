package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type defaultUlimitBenchmark struct{}

func (c *defaultUlimitBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.7",
			Description: "Ensure the default ulimit is configured appropriately",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *defaultUlimitBenchmark) Run() (result storage.BenchmarkCheckResult) {
	if _, ok := utils.DockerConfig["default-ulimit"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "No default-ulimit values are set")
		return
	}
	utils.Pass(&result)
	return
}

// NewDefaultUlimitBenchmark implements CIS-2.7
func NewDefaultUlimitBenchmark() utils.Check {
	return &defaultUlimitBenchmark{}
}
