package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type pidCgroupBenchmark struct{}

func (c *pidCgroupBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.28",
			Description: "Ensure PIDs cgroup limit is used",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *pidCgroupBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.PidsLimit <= 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) does not have pids limit set", container.ID, container.Name)
		}
	}
	return
}

// NewPidCgroupBenchmark implements CIS-5.28
func NewPidCgroupBenchmark() utils.Check {
	return &pidCgroupBenchmark{}
}
