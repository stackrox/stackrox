package containerruntime

import (
	"strconv"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type privilegedPortsBenchmark struct{}

func (c *privilegedPortsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.7",
			Description: "Ensure privileged ports are not mapped within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *privilegedPortsBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				portNum, err := strconv.Atoi(binding.HostPort)
				if err != nil {
					utils.Warn(&result)
					utils.AddNotef(&result, "Could not parse host port for container '%v': '%v'", container.ID, binding.HostPort)
					continue
				}
				if portNum < 1024 {
					utils.Warn(&result)
					utils.AddNotef(&result, "Container '%v' binds '%v' to privileged host port '%v'", container.ID, containerPort, portNum)
				}
			}
		}
	}
	return
}

// NewPrivilegedPortsBenchmark implements CIS-5.7
func NewPrivilegedPortsBenchmark() utils.Check {
	return &privilegedPortsBenchmark{}
}
