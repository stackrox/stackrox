package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type liveRestoreEnabledBenchmark struct{}

func (c *liveRestoreEnabledBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.14",
		Description:  "Ensure live restore is Enabled",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *liveRestoreEnabledBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Result = common.Warn
		result.AddNotes(err.Error())
		return
	}
	if !info.LiveRestoreEnabled {
		result.Result = common.Warn
		result.AddNotes("Live restore is not enabled")
		return
	}
	result.Result = common.Pass
	return
}

// NewLiveRestoreEnabledBenchmark implements CIS-2.14
func NewLiveRestoreEnabledBenchmark() common.Benchmark {
	return &liveRestoreEnabledBenchmark{}
}
