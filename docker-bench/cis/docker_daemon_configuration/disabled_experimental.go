package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type disableExperimentalBenchmark struct{}

func (c *disableExperimentalBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.17",
		Description:  "Ensure experimental features are avoided in production",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *disableExperimentalBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Warn()
		result.AddNotes(err.Error())
		return
	}
	if info.ExperimentalBuild {
		result.Warn()
		result.AddNotes("Docker is running in experimental mode")
		return
	}
	result.Pass()
	return
}

// NewDisableExperimentalBenchmark implements CIS-2.17
func NewDisableExperimentalBenchmark() common.Benchmark {
	return &disableExperimentalBenchmark{}
}
