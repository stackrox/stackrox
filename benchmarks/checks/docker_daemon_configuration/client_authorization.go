package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type authorizationPluginBenchmark struct{}

func (c *authorizationPluginBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.11",
			Description: "Ensure that authorization for Docker client commands is enabled",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *authorizationPluginBenchmark) Run() (result v1.CheckResult) {
	_, ok := utils.DockerConfig["authorization-plugin"]
	if !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "No authorization plugin is enabled for the docker client")
		return
	}
	// TODO(cgorman) search for image?
	utils.Pass(&result)
	return
}

// NewAuthorizationPluginBenchmark implements CIS-2.11
func NewAuthorizationPluginBenchmark() utils.Check {
	return &authorizationPluginBenchmark{}
}
