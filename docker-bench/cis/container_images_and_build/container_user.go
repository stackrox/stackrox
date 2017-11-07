package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type containerUserBenchmark struct{}

func (c *containerUserBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.1",
		Description:  "Ensure a user for the container has been created",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *containerUserBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.Config.User == "" || container.Config.User == "root" {
			result.Warn()
			result.AddNotef("Container %v with image %v is running as root user", container.ID, container.Config.Image)
		}
	}
	return
}

// NewContainerUserBenchmark implements CIS-4.1
func NewContainerUserBenchmark() common.Benchmark {
	return &containerUserBenchmark{}
}
