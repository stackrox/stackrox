package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type usernsBenchmark struct{}

func (c *usernsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.30",
		Description:  "Ensure the host's user namespaces is not shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *usernsBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.UsernsMode.IsHost() {
			result.Warn()
			result.AddNotef("Container %v has user namespace set to host", container.ID)
		}
	}
	return
}

// NewUsernsBenchmark implements CIS-5.30
func NewUsernsBenchmark() common.Benchmark {
	return &usernsBenchmark{}
}
