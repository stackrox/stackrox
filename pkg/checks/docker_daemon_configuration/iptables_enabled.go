package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type ipTablesEnabledBenchmark struct{}

func (c *ipTablesEnabledBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.3",
			Description: "Ensure Docker is allowed to make changes to iptables",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *ipTablesEnabledBenchmark) Run() (result v1.CheckResult) {
	values, ok := utils.DockerConfig["iptables"]
	if !ok {
		utils.Pass(&result)
		return
	}
	if values.Matches("false") {
		utils.Warn(&result)
		utils.AddNotes(&result, "Docker is not configured to modify iptables")
		return
	}
	utils.Pass(&result)
	return
}

// NewIPTablesEnabledBenchmark implements CIS-2.3
func NewIPTablesEnabledBenchmark() utils.Check {
	return &ipTablesEnabledBenchmark{}
}
