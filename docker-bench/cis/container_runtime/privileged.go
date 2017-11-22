package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type privilegedBenchmark struct{}

func (c *privilegedBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.4",
			Description: "Ensure privileged containers are not used",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *privilegedBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.Privileged {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v is running as privileged", container.ID)
		}
	}
	return
}

// NewPrivilegedBenchmark implements CIS-5.4
func NewPrivilegedBenchmark() utils.Benchmark {
	return &privilegedBenchmark{}
}
