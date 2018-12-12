package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type ipTablesEnabledBenchmark struct{}

func (c *ipTablesEnabledBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.3",
			Description: "Ensure Docker is allowed to make changes to iptables",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *ipTablesEnabledBenchmark) Run() (result storage.BenchmarkCheckResult) {
	values, ok := utils.DockerConfig["iptables"]
	if !ok {
		utils.Pass(&result)
		return
	}
	if values.Matches("false") {
		utils.Warn(&result)
		utils.AddNotes(&result, "Docker is not configured to modify iptables")
		return
	}
	utils.Pass(&result)
	return
}

// NewIPTablesEnabledBenchmark implements CIS-2.3
func NewIPTablesEnabledBenchmark() utils.Check {
	return &ipTablesEnabledBenchmark{}
}
