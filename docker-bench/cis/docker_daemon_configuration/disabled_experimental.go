package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type disableExperimentalBenchmark struct{}

func (c *disableExperimentalBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.17",
			Description: "Ensure experimental features are avoided in production",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *disableExperimentalBenchmark) Run() (result v1.CheckResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	if info.ExperimentalBuild {
		utils.Warn(&result)
		utils.AddNotes(&result, "Docker is running in experimental mode")
		return
	}
	utils.Pass(&result)
	return
}

// NewDisableExperimentalBenchmark implements CIS-2.17
func NewDisableExperimentalBenchmark() utils.Check {
	return &disableExperimentalBenchmark{}
}
