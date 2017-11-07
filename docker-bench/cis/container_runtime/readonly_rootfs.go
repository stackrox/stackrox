package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type readonlyRootfsBenchmark struct{}

func (c *readonlyRootfsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.12",
		Description:  "Ensure the container's root filesystem is mounted as read only",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *readonlyRootfsBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if !container.HostConfig.ReadonlyRootfs {
			result.Warn()
			result.AddNotef("Container %v does not have a readonly rootfs", container.ID)
		}
	}
	return
}

// NewReadonlyRootfsBenchmark implements CIS-5.12
func NewReadonlyRootfsBenchmark() common.Benchmark {
	return &readonlyRootfsBenchmark{}
}
