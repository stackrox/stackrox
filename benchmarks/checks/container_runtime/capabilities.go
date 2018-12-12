package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type capabilitiesBenchmark struct{}

func (c *capabilitiesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.3",
			Description: "Ensure Linux Kernel Capabilities are restricted within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *capabilitiesBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Info(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.CapAdd) > 0 {
			utils.AddNotef(&result, "Container '%s' (%s) adds capabilities: %v", container.ID, container.Name, strings.Join(container.HostConfig.CapAdd, ","))
			continue
		}
	}
	return
}

// NewCapabilitiesBenchmark implements CIS-5.3
func NewCapabilitiesBenchmark() utils.Check {
	return &capabilitiesBenchmark{}
}
