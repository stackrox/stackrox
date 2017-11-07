package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type ipcNamespaceBenchmark struct{}

func (c *ipcNamespaceBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.16",
		Description:  "Ensure the host's IPC namespace is not shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *ipcNamespaceBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.IpcMode.IsHost() {
			result.Warn()
			result.AddNotef("Container %v has ipc mode set to host", container.ID)
		}
	}
	return
}

// NewIpcNamespaceBenchmark implements CIS-5.16
func NewIpcNamespaceBenchmark() common.Benchmark {
	return &ipcNamespaceBenchmark{}
}
