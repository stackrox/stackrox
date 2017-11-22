package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/docker/docker/api/types/mount"
)

type mountPropagationBenchmark struct{}

func (c *mountPropagationBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.19",
			Description: "Ensure mount propagation mode is not set to shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *mountPropagationBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, containerMount := range container.Mounts {
			if containerMount.Propagation == mount.PropagationShared {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container %v and mount %v uses shared propagation", container.ID, containerMount.Name)
			}
		}
	}
	return
}

// NewMountPropagationBenchmark implements CIS-5.19
func NewMountPropagationBenchmark() utils.Benchmark {
	return &mountPropagationBenchmark{}
}
