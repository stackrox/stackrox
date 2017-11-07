package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
	"github.com/docker/docker/api/types/mount"
)

type mountPropagationBenchmark struct{}

func (c *mountPropagationBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.19",
		Description:  "Ensure mount propagation mode is not set to shared",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *mountPropagationBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for _, containerMount := range container.Mounts {
			if containerMount.Propagation == mount.PropagationShared {
				result.Warn()
				result.AddNotef("Container %v and mount %v uses shared propagation", container.ID, containerMount.Name)
			}
		}
	}
	return
}

// NewMountPropagationBenchmark implements CIS-5.19
func NewMountPropagationBenchmark() common.Benchmark {
	return &mountPropagationBenchmark{}
}
