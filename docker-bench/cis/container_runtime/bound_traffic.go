package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type specificHostInterfaceBenchmark struct{}

func (c *specificHostInterfaceBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.13",
		Description:  "Ensure incoming container traffic is binded to a specific host interface",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *specificHostInterfaceBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				if strings.Contains(binding.HostIP, "0.0.0.0") {
					result.Warn()
					result.AddNotef("Container %v binds %v -> 0.0.0.0 %v", container.ID, containerPort, binding.HostPort)
				}
			}
		}
	}
	return
}

// NewSpecificHostInterfaceBenchmark implements CIS-5.13
func NewSpecificHostInterfaceBenchmark() common.Benchmark {
	return &specificHostInterfaceBenchmark{}
}
