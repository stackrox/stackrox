package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type disableExperimentalBenchmark struct{}

func (c *disableExperimentalBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.17",
			Description: "Ensure experimental features are avoided in production",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *disableExperimentalBenchmark) Run() (result v1.CheckResult) {
	if utils.DockerInfo.ExperimentalBuild {
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
