package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type acquiringPrivilegesBenchmark struct{}

func (c *acquiringPrivilegesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.25",
		Description:  "Ensure the container is restricted from acquiring additional privileges",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *acquiringPrivilegesBenchmark) Run() (result common.TestResult) {
	result.Pass()
LOOP:
	for _, container := range common.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(opt, "no-new-privileges") {
				continue LOOP
			}
		}
		result.Warn()
		result.AddNotef("Container %v does not set no-new-privileges in security opts", container.ID)
	}
	return
}

// NewAcquiringPrivilegesBenchmark implements CIS-5.25
func NewAcquiringPrivilegesBenchmark() common.Benchmark {
	return &acquiringPrivilegesBenchmark{}
}
