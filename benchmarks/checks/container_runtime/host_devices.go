package containerruntime

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type hostDevicesBenchmark struct{}

func (c *hostDevicesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.17",
			Description: "Ensure host devices are not directly exposed to containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *hostDevicesBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if len(container.HostConfig.Devices) > 0 {
			utils.Warn(&result)
			devices := make([]string, 0, len(container.HostConfig.Devices))
			for _, device := range container.HostConfig.Devices {
				devices = append(devices, fmt.Sprintf("%v:%v", device.PathOnHost, device.PathInContainer))
			}
			utils.AddNotef(&result, "Container '%v' (%v) has host devices %+v exposed to it", container.ID, container.Name, strings.Join(devices, ","))
		}
	}
	return
}

// NewHostDevicesBenchmark implements CIS-5.17
func NewHostDevicesBenchmark() utils.Check {
	return &hostDevicesBenchmark{}
}
