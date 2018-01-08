package dockerdaemonconfiguration

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type remoteLoggingBenchmark struct{}

func (c *remoteLoggingBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.12",
			Description: "Ensure centralized and remote logging is configured",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *remoteLoggingBenchmark) Run() (result v1.CheckResult) {
	if utils.DockerInfo.LoggingDriver == "json-file" {
		utils.Warn(&result)
		utils.AddNotes(&result,
			fmt.Sprintf("Logging driver '%v' is currently not configured for remote logging", utils.DockerInfo.LoggingDriver))
		return
	}
	utils.Pass(&result)
	return
}

// NewRemoteLoggingBenchmark implements CIS-2.12
func NewRemoteLoggingBenchmark() utils.Check {
	return &remoteLoggingBenchmark{}
}
