package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
	"github.com/docker/docker/api/types/mount"
)

type mountPropagationBenchmark struct{}

func (c *mountPropagationBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.19",
			Description: "Ensure mount propagation mode is not set to shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *mountPropagationBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, containerMount := range container.Mounts {
			if containerMount.Propagation == mount.PropagationShared {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container '%v' and mount '%v' uses shared propagation", container.ID, containerMount.Name)
			}
		}
	}
	return
}

// NewMountPropagationBenchmark implements CIS-5.19
func NewMountPropagationBenchmark() utils.Check {
	return &mountPropagationBenchmark{}
}
