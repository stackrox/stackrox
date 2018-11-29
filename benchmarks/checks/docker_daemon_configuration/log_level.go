package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type logLevelBenchmark struct{}

func (c *logLevelBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.2",
			Description: "Ensure the logging level is set to 'info'",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *logLevelBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if vals, ok := utils.DockerConfig["log-level"]; ok {
		if _, exists := vals.Contains("info"); !exists {
			utils.Warn(&result)
			utils.AddNotef(&result, "log-level is set to '%v'", vals[0])
			return
		}
	}
	utils.Pass(&result)
	return
}

// NewLogLevelBenchmark implements CIS-2.2
func NewLogLevelBenchmark() utils.Check {
	return &logLevelBenchmark{}
}
