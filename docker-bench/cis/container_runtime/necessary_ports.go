package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type necessaryPortsBenchmark struct{}

func (c *necessaryPortsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.8",
		Description:  "Ensure only needed ports are open on the container",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *necessaryPortsBenchmark) Run() (result common.TestResult) {
	result.Note()
	for _, container := range common.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				result.AddNotef("Container %v binds container %v -> host %v", container.ID, containerPort, binding.HostPort)
			}
		}
	}
	return
}

// NewNecessaryPortsBenchmark implements CIS-5.8
func NewNecessaryPortsBenchmark() common.Benchmark {
	return &necessaryPortsBenchmark{}
}
