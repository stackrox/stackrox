package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type restrictContainerPrivilegesBenchmark struct{}

func (c *restrictContainerPrivilegesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.18",
			Description: "Ensure containers are restricted from acquiring new privileges",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *restrictContainerPrivilegesBenchmark) Run() (result v1.BenchmarkTestResult) {
	if opts, ok := utils.DockerConfig["no-new-privileges"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "ContainersRunning are not prevented from acquiring new privileges by default")
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
func NewRestrictContainerPrivilegesBenchmark() utils.Benchmark {
	return &restrictContainerPrivilegesBenchmark{}
}
