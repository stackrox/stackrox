package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type disableLegacyRegistryBenchmark struct{}

func (c *disableLegacyRegistryBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.13",
			Description: "Ensure operations on legacy registry (v1) are Disabled",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *disableLegacyRegistryBenchmark) Run() (result v1.CheckResult) {
	if _, ok := utils.DockerConfig["disable-legacy-registry"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "Legacy registry is not disabled")
		return
	}
	utils.Pass(&result)
	return
}

// NewDisableLegacyRegistryBenchmark implements CIS-2.13
func NewDisableLegacyRegistryBenchmark() utils.Check {
	return &disableLegacyRegistryBenchmark{}
}
