package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type liveRestoreEnabledBenchmark struct{}

func (c *liveRestoreEnabledBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.14",
			Description: "Ensure live restore is Enabled",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *liveRestoreEnabledBenchmark) Run() (result v1.BenchmarkTestResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	if !info.LiveRestoreEnabled {
		utils.Warn(&result)
		utils.AddNotes(&result, "Live restore is not enabled")
		return
	}
	utils.Pass(&result)
	return
}

// NewLiveRestoreEnabledBenchmark implements CIS-2.14
func NewLiveRestoreEnabledBenchmark() utils.Benchmark {
	return &liveRestoreEnabledBenchmark{}
}
