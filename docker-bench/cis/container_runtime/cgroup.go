package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type cgroupBenchmark struct{}

func (c *cgroupBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.24",
			Description: "Ensure cgroup usage is confirmed",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *cgroupBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.CgroupParent != "docker" && container.HostConfig.CgroupParent != "" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v has the cgroup parent set to %v", container.ID, container.HostConfig.CgroupParent)
		}
	}
	return
}

// NewCgroupBenchmark implements CIS-5.24
func NewCgroupBenchmark() utils.Benchmark {
	return &cgroupBenchmark{}
}
