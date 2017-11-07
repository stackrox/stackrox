package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type utsNamespaceBenchmark struct{}

func (c *utsNamespaceBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.20",
		Description:  "Ensure the host's UTS namespace is not shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *utsNamespaceBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.HostConfig.UTSMode.IsHost() {
			result.Warn()
			result.AddNotef("Container %v has UTS mode set to host", container.ID)
		}
	}
	return
}

// NewUTSNamespaceBenchmark implements CIS-5.20
func NewUTSNamespaceBenchmark() common.Benchmark {
	return &utsNamespaceBenchmark{}
}
