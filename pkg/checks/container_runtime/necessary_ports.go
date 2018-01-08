package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type necessaryPortsBenchmark struct{}

func (c *necessaryPortsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.8",
			Description: "Ensure only needed ports are open on the container",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *necessaryPortsBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	for _, container := range utils.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				utils.AddNotef(&result, "Container '%v' binds container '%v' -> host '%v'", container.ID, containerPort, binding.HostPort)
			}
		}
	}
	return
}

// NewNecessaryPortsBenchmark implements CIS-5.8
func NewNecessaryPortsBenchmark() utils.Check {
	return &necessaryPortsBenchmark{}
}
