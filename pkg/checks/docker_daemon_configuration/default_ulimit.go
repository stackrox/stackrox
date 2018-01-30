package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type defaultUlimitBenchmark struct{}

func (c *defaultUlimitBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.7",
			Description: "Ensure the default ulimit is configured appropriately",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *defaultUlimitBenchmark) Run() (result v1.CheckResult) {
	if _, ok := utils.DockerConfig["default-ulimit"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "No default-ulimit values are set")
		return
	}
	utils.Pass(&result)
	return
}

// NewDefaultUlimitBenchmark implements CIS-2.7
func NewDefaultUlimitBenchmark() utils.Check {
	return &defaultUlimitBenchmark{}
}
