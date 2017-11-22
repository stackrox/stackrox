package dockerdaemonconfiguration

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type remoteLoggingBenchmark struct{}

func (c *remoteLoggingBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.12",
			Description: "Ensure centralized and remote logging is configured",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *remoteLoggingBenchmark) Run() (result v1.BenchmarkTestResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	if info.LoggingDriver == "json-file" {
		utils.Warn(&result)
		utils.AddNotes(&result, fmt.Sprintf("Logging driver %v is currently not configured for remote logging", info.LoggingDriver))
		return
	}
	utils.Pass(&result)
	return
}

// NewRemoteLoggingBenchmark implements CIS-2.12
func NewRemoteLoggingBenchmark() utils.Benchmark {
	return &remoteLoggingBenchmark{}
}
