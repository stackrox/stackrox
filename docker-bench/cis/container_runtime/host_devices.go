package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type hostDevicesBenchmark struct{}

func (c *hostDevicesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.17",
			Description: "Ensure host devices are not directly exposed to containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *hostDevicesBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.Devices) > 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v has host devices %+v exposed to it", container.ID, container.HostConfig.Devices)
		}
	}
	return
}

// NewHostDevicesBenchmark implements CIS-5.17
func NewHostDevicesBenchmark() utils.Benchmark {
	return &hostDevicesBenchmark{}
}
