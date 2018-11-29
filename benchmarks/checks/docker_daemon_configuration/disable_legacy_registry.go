package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type disableLegacyRegistryBenchmark struct{}

func (c *disableLegacyRegistryBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.13",
			Description: "Ensure operations on legacy registry (v1) are Disabled",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *disableLegacyRegistryBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if _, ok := utils.DockerConfig["disable-legacy-registry"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "Legacy registry is not disabled")
		return
	}
	utils.Pass(&result)
	return
}

// NewDisableLegacyRegistryBenchmark implements CIS-2.13
func NewDisableLegacyRegistryBenchmark() utils.Check {
	return &disableLegacyRegistryBenchmark{}
}
