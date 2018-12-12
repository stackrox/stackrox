package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type sharedNetworkBenchmark struct{}

func (c *sharedNetworkBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.9",
			Description: "Ensure the host's network namespace is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *sharedNetworkBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.NetworkMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has network set to --net=host", container.ID, container.Name)
		}
	}
	return
}

// NewSharedNetworkBenchmark implements CIS-5.9
func NewSharedNetworkBenchmark() utils.Check {
	return &sharedNetworkBenchmark{}
}
