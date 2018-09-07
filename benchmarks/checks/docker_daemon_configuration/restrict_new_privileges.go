package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type restrictContainerPrivilegesBenchmark struct{}

func (c *restrictContainerPrivilegesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.18",
			Description: "Ensure containers are restricted from acquiring new privileges",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *restrictContainerPrivilegesBenchmark) Run() (result v1.CheckResult) {
	if opts, ok := utils.DockerConfig["no-new-privileges"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "Running containers are not prevented from acquiring new privileges by default")
		return
	} else if opts.Matches("false") {
		utils.Warn(&result)
		utils.AddNotes(&result, "no-new-privileges is not set to false")
		return
	}
	utils.Pass(&result)
	return
}

// NewRestrictContainerPrivilegesBenchmark implements CIS-2.18
func NewRestrictContainerPrivilegesBenchmark() utils.Check {
	return &restrictContainerPrivilegesBenchmark{}
}
