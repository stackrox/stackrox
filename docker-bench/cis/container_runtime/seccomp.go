package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type seccompBenchmark struct{}

func (c *seccompBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.21",
		Description:  "Ensure the default seccomp profile is not Disabled",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *seccompBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(opt, "seccomp:unconfined") {
				result.Warn()
				result.AddNotef("Container %v has seccomp set to unconfined", container.ID)
				break
			}
		}
	}
	return
}

// NewSeccompBenchmark implements CIS-5.21
func NewSeccompBenchmark() common.Benchmark {
	return &seccompBenchmark{}
}
