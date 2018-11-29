package dockerdaemonconfiguration

import (
	"fmt"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type remoteLoggingBenchmark struct{}

func (c *remoteLoggingBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.12",
			Description: "Ensure centralized and remote logging is configured",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *remoteLoggingBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if utils.DockerInfo.LoggingDriver == "json-file" {
		utils.Warn(&result)
		utils.AddNotes(&result,
			fmt.Sprintf("Logging driver '%v' is currently not configured for remote logging", utils.DockerInfo.LoggingDriver))
		return
	}
	utils.Pass(&result)
	return
}

// NewRemoteLoggingBenchmark implements CIS-2.12
func NewRemoteLoggingBenchmark() utils.Check {
	return &remoteLoggingBenchmark{}
}
