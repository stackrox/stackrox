package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
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

func newExpectedCapDrop() map[string]bool {
	return map[string]bool{
		"NET_ADMIN":  false,
		"SYS_ADMIN":  false,
		"SYS_MODULE": false,
	}
}

func (c *capabilitiesBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.CapAdd) > 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' adds capabilities: %v", container.ID, strings.Join(container.HostConfig.CapAdd, ","))
			continue
		}
		capDropMap := newExpectedCapDrop()
		for _, drop := range container.HostConfig.CapDrop {
			capDropMap[drop] = true
		}
		for k, v := range capDropMap {
			if !v {
				utils.Warn(&result)
				utils.AddNotef(&result, "Expected container '%v' to drop capability '%v'", container.ID, k)
			}
		}
	}
	return
}

// NewCapabilitiesBenchmark implements CIS-5.3
func NewCapabilitiesBenchmark() utils.Check {
	return &capabilitiesBenchmark{}
}
