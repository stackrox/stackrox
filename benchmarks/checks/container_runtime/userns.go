package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type usernsBenchmark struct{}

func (c *usernsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.30",
			Description: "Ensure the host's user namespaces is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *usernsBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.UsernsMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has user namespace set to host", container.ID, container.Name)
		}
	}
	return
}

// NewUsernsBenchmark implements CIS-5.30
func NewUsernsBenchmark() utils.Check {
	return &usernsBenchmark{}
}
