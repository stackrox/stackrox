package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type seccompBenchmark struct{}

func (c *seccompBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.21",
			Description: "Ensure the default seccomp profile is not Disabled",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *seccompBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(opt, "seccomp:unconfined") {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container '%v' (%v) has seccomp set to unconfined", container.ID, container.Name)
				break
			}
		}
	}
	return
}

// NewSeccompBenchmark implements CIS-5.21
func NewSeccompBenchmark() utils.Check {
	return &seccompBenchmark{}
}
