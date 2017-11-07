package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type bridgeNetworkBenchmark struct{}

func (c *bridgeNetworkBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.29",
		Description:  "Ensure Docker's default bridge docker0 is not used",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *bridgeNetworkBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if _, ok := container.NetworkSettings.Networks["bridge"]; ok {
			result.Warn()
			result.AddNotef("Container %v is running on the bridge network", container.ID)
		}
	}
	return
}

// NewBridgeNetworkBenchmark implements CIS-5.29
func NewBridgeNetworkBenchmark() common.Benchmark {
	return &bridgeNetworkBenchmark{}
}
