package containerruntime

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type readonlyRootfsBenchmark struct{}

func (c *readonlyRootfsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.12",
			Description: "Ensure the container's root filesystem is mounted as read only",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *readonlyRootfsBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if !container.HostConfig.ReadonlyRootfs {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' (%v) does not have a readonly rootfs", container.ID, container.Name)
		}
	}
	return
}

// NewReadonlyRootfsBenchmark implements CIS-5.12
func NewReadonlyRootfsBenchmark() utils.Check {
	return &readonlyRootfsBenchmark{}
}
