package dockerdaemonconfiguration

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type remoteLoggingBenchmark struct{}

func (c *remoteLoggingBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.12",
		Description:  "Ensure centralized and remote logging is configured",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *remoteLoggingBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Result = common.Warn
		result.AddNotes(err.Error())
		return
	}
	if info.LoggingDriver == "json-file" {
		result.Result = common.Warn
		result.AddNotes(fmt.Sprintf("Logging driver %v is currently not configured for remote logging", info.LoggingDriver))
		return
	}
	result.Result = common.Pass
	return
}

// NewRemoteLoggingBenchmark implements CIS-2.12
func NewRemoteLoggingBenchmark() common.Benchmark {
	return &remoteLoggingBenchmark{}
}
