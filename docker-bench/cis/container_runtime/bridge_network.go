package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type bridgeNetworkBenchmark struct{}

func (c *bridgeNetworkBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.29",
			Description: "Ensure Docker's default bridge docker0 is not used",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *bridgeNetworkBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if _, ok := container.NetworkSettings.Networks["bridge"]; ok {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v is running on the bridge network", container.ID)
		}
	}
	return
}

// NewBridgeNetworkBenchmark implements CIS-5.29
func NewBridgeNetworkBenchmark() utils.Benchmark {
	return &bridgeNetworkBenchmark{}
}
