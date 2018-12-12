package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type ipcNamespaceBenchmark struct{}

func (c *ipcNamespaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.16",
			Description: "Ensure the host's IPC namespace is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *ipcNamespaceBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.IpcMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has ipc mode set to host", container.ID, container.Name)
		}
	}
	return
}

// NewIpcNamespaceBenchmark implements CIS-5.16
func NewIpcNamespaceBenchmark() utils.Check {
	return &ipcNamespaceBenchmark{}
}
