package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type restrictContainerPrivilegesBenchmark struct{}

func (c *restrictContainerPrivilegesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.18",
		Description:  "Ensure containers are restricted from acquiring new privileges",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *restrictContainerPrivilegesBenchmark) Run() (result common.TestResult) {
	if opts, ok := common.DockerConfig["no-new-privileges"]; !ok {
		result.Warn()
		result.AddNotes("ContainersRunning are not prevented from acquiring new privileges by default")
		return
	} else if opts.Matches("false") {
		result.Warn()
		result.AddNotes("no-new-privileges is not set to false")
		return
	}
	result.Pass()
	return
}

// NewRestrictContainerPrivilegesBenchmark implements CIS-2.18
func NewRestrictContainerPrivilegesBenchmark() common.Benchmark {
	return &restrictContainerPrivilegesBenchmark{}
}
