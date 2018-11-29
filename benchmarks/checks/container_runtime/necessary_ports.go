package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type necessaryPortsBenchmark struct{}

func (c *necessaryPortsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.8",
			Description: "Ensure only needed ports are open on the container",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *necessaryPortsBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	for _, container := range utils.ContainersRunning {
		for containerPort, hostBinding := range container.NetworkSettings.Ports {
			for _, binding := range hostBinding {
				utils.AddNotef(&result, "Container '%v' (%v) binds container '%v' -> host '%v'", container.ID, container.Name, containerPort, binding.HostPort)
			}
		}
	}
	return
}

// NewNecessaryPortsBenchmark implements CIS-5.8
func NewNecessaryPortsBenchmark() utils.Check {
	return &necessaryPortsBenchmark{}
}
