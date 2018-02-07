package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type utsNamespaceBenchmark struct{}

func (c *utsNamespaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.20",
			Description: "Ensure the host's UTS namespace is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *utsNamespaceBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.UTSMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) has UTS mode set to host", container.ID, container.Name)
		}
	}
	return
}

// NewUTSNamespaceBenchmark implements CIS-5.20
func NewUTSNamespaceBenchmark() utils.Check {
	return &utsNamespaceBenchmark{}
}
