package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type bridgeNetworkBenchmark struct{}

func (c *bridgeNetworkBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.29",
			Description: "Ensure Docker's default bridge docker0 is not used",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *bridgeNetworkBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if _, ok := container.NetworkSettings.Networks["bridge"]; ok {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) is running on the bridge network", container.ID, container.Name)
		}
	}
	return
}

// NewBridgeNetworkBenchmark implements CIS-5.29
func NewBridgeNetworkBenchmark() utils.Check {
	return &bridgeNetworkBenchmark{}
}
