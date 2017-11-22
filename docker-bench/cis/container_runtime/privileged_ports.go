package containerruntime

import (
	"strconv"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type privilegedPortsBenchmark struct{}

func (c *privilegedPortsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.7",
			Description: "Ensure privileged ports are not mapped within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *privilegedPortsBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				portNum, err := strconv.Atoi(binding.HostPort)
				if err != nil {
					utils.Warn(&result)
					utils.AddNotef(&result, "Could not parse host port for container %v: %+v", container.ID, binding.HostPort)
					continue
				}
				if portNum < 1024 {
					utils.Warn(&result)
					utils.AddNotef(&result, "Container %v binds %v to privileged host port %v", containerPort, container.ID, portNum)
				}
			}
		}
	}
	return
}

// NewPrivilegedPortsBenchmark implements CIS-5.7
func NewPrivilegedPortsBenchmark() utils.Benchmark {
	return &privilegedPortsBenchmark{}
}
