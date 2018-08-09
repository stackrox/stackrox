package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type capabilitiesBenchmark struct{}

func (c *capabilitiesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.3",
			Description: "Ensure Linux Kernel Capabilities are restricted within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *capabilitiesBenchmark) Run() (result v1.CheckResult) {
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
