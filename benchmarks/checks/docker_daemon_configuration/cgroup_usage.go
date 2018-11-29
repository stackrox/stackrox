package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type cgroupUsageBenchmark struct{}

func (c *cgroupUsageBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.9",
			Description: "Ensure the default cgroup usage has been confirmed",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *cgroupUsageBenchmark) Run() (result v1.BenchmarkCheckResult) {
	if parent, ok := utils.DockerConfig["cgroup-parent"]; ok {
		utils.Warn(&result)
		utils.AddNotef(&result, "Cgroup path is set as '%v'", parent)
		return
	}
	utils.Pass(&result)
	return
}

// NewCgroupUsageBenchmark implements CIS-2.9
func NewCgroupUsageBenchmark() utils.Check {
	return &cgroupUsageBenchmark{}
}
