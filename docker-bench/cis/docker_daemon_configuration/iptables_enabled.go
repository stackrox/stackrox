package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type ipTablesEnabledBenchmark struct{}

func (c *ipTablesEnabledBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.3",
			Description: "Ensure Docker is allowed to make changes to iptables",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *ipTablesEnabledBenchmark) Run() (result v1.BenchmarkTestResult) {
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
func NewIPTablesEnabledBenchmark() utils.Benchmark {
	return &ipTablesEnabledBenchmark{}
}
