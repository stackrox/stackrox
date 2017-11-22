package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type defaultUlimitBenchmark struct{}

func (c *defaultUlimitBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.7",
			Description: "Ensure the default ulimit is configured appropriately",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *defaultUlimitBenchmark) Run() (result v1.BenchmarkTestResult) {
	if _, ok := utils.DockerConfig["default-ulimit"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "No default-ulimit values are set")
		return
	}
	utils.Pass(&result)
	return
}

// NewDefaultUlimitBenchmark implements CIS-2.7
func NewDefaultUlimitBenchmark() utils.Benchmark {
	return &defaultUlimitBenchmark{}
}
