package containerruntime

import (
	"strconv"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type privilegedPortsBenchmark struct{}

func (c *privilegedPortsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.7",
		Description:  "Ensure privileged ports are not mapped within containers",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *privilegedPortsBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				portNum, err := strconv.Atoi(binding.HostPort)
				if err != nil {
					result.Warn()
					result.AddNotef("Could not parse host port for container %v: %+v", container.ID, binding.HostPort)
					continue
				}
				if portNum < 1024 {
					result.Warn()
					result.AddNotef("Container %v binds %v to privileged host port %v", containerPort, container.ID, portNum)
				}
			}
		}
	}
	return
}

// NewPrivilegedPortsBenchmark implements CIS-5.7
func NewPrivilegedPortsBenchmark() common.Benchmark {
	return &privilegedPortsBenchmark{}
}
