package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type liveRestoreEnabledBenchmark struct{}

func (c *liveRestoreEnabledBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.14",
			Description: "Ensure live restore is Enabled",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *liveRestoreEnabledBenchmark) Run() (result v1.CheckResult) {
	if !utils.DockerInfo.LiveRestoreEnabled {
		utils.Warn(&result)
		utils.AddNotes(&result, "Live restore is not enabled")
		return
	}
	utils.Pass(&result)
	return
}

// NewLiveRestoreEnabledBenchmark implements CIS-2.14
func NewLiveRestoreEnabledBenchmark() utils.Check {
	return &liveRestoreEnabledBenchmark{}
}
