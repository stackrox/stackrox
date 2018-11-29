package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type cgroupBenchmark struct{}

func (c *cgroupBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.24",
			Description: "Ensure cgroup usage is confirmed",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *cgroupBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.CgroupParent != "docker" && container.HostConfig.CgroupParent != "" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has the cgroup parent set to '%v'", container.ID, container.Name, container.HostConfig.CgroupParent)
		}
	}
	return
}

// NewCgroupBenchmark implements CIS-5.24
func NewCgroupBenchmark() utils.Check {
	return &cgroupBenchmark{}
}
