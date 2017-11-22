package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type acquiringPrivilegesBenchmark struct{}

func (c *acquiringPrivilegesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.25",
			Description: "Ensure the container is restricted from acquiring additional privileges",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *acquiringPrivilegesBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
LOOP:
	for _, container := range utils.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(opt, "no-new-privileges") {
				continue LOOP
			}
		}
		utils.Warn(&result)
		utils.AddNotef(&result, "Container %v does not set no-new-privileges in security opts", container.ID)
	}
	return
}

// NewAcquiringPrivilegesBenchmark implements CIS-5.25
func NewAcquiringPrivilegesBenchmark() utils.Benchmark {
	return &acquiringPrivilegesBenchmark{}
}
