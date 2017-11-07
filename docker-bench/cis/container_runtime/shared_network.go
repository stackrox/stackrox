package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type sharedNetworkBenchmark struct{}

func (c *sharedNetworkBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.9",
		Description:  "Ensure the host's network namespace is not shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *sharedNetworkBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.NetworkMode.IsHost() {
			result.Warn()
			result.AddNotef("Container %v has network set to --net=host", container.ID)
		}
	}
	return
}

// NewSharedNetworkBenchmark implements CIS-5.9
func NewSharedNetworkBenchmark() common.Benchmark {
	return &sharedNetworkBenchmark{}
}
