package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type specificHostInterfaceBenchmark struct{}

func (c *specificHostInterfaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.13",
			Description: "Ensure incoming container traffic is binded to a specific host interface",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *specificHostInterfaceBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				if strings.Contains(binding.HostIP, "0.0.0.0") {
					utils.Warn(&result)
					utils.AddNotef(&result, "Container '%v' (%v) binds '%v' -> '0.0.0.0 %v'", container.ID, container.Name, containerPort, binding.HostPort)
				}
			}
		}
	}
	return
}

// NewSpecificHostInterfaceBenchmark implements CIS-5.13
func NewSpecificHostInterfaceBenchmark() utils.Check {
	return &specificHostInterfaceBenchmark{}
}
