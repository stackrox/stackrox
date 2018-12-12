package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type pidNamespaceBenchmark struct{}

func (c *pidNamespaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.15",
			Description: "Ensure the host's process namespace is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *pidNamespaceBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.PidMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has pid mode set to host", container.ID, container.Name)
		}
	}
	return
}

// NewPidNamespaceBenchmark implements CIS-5.15
func NewPidNamespaceBenchmark() utils.Check {
	return &pidNamespaceBenchmark{}
}
