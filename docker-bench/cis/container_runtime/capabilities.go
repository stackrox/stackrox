package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type capabilitiesBenchmark struct{}

func (c *capabilitiesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.3",
		Description:  "Ensure Linux Kernel Capabilities are restricted within containers",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func newExpectedCapDrop() map[string]bool {
	return map[string]bool{
		"NET_ADMIN":  false,
		"SYS_ADMIN":  false,
		"SYS_MODULE": false,
	}
}

func (c *capabilitiesBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if len(container.HostConfig.CapAdd) > 0 {
			result.Warn()
			result.AddNotef("Container %v adds capabilities: %+v", container.ID, container.HostConfig.CapAdd)
			continue
		}
		capDropMap := newExpectedCapDrop()
		for _, drop := range container.HostConfig.CapDrop {
			capDropMap[drop] = true
		}
		for k, v := range capDropMap {
			if !v {
				result.Warn()
				result.AddNotef("Expected container %v to drop capability %v", container.ID, k)
			}
		}
	}
	return
}

// NewCapabilitiesBenchmark implements CIS-5.3
func NewCapabilitiesBenchmark() common.Benchmark {
	return &capabilitiesBenchmark{}
}
