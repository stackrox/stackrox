package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type hostDevicesBenchmark struct{}

func (c *hostDevicesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.17",
		Description:  "Ensure host devices are not directly exposed to containers",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *hostDevicesBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if len(container.HostConfig.Devices) > 0 {
			result.Warn()
			result.AddNotef("Container %v has host devices %+v exposed to it", container.ID, container.HostConfig.Devices)
		}
	}
	return
}

// NewHostDevicesBenchmark implements CIS-5.17
func NewHostDevicesBenchmark() common.Benchmark {
	return &hostDevicesBenchmark{}
}
