package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type dockerSocketMountBenchmark struct{}

func (c *dockerSocketMountBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.31",
			Description: "Ensure the Docker socket is not mounted inside any containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *dockerSocketMountBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, containerMount := range container.Mounts {
			if strings.Contains(containerMount.Source, "docker.sock") {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container '%v' (%v) has mounted docker.sock", container.ID, container.Name)
			}
		}
	}
	return
}

// NewDockerSocketMountBenchmark implements CIS-5.31
func NewDockerSocketMountBenchmark() utils.Check {
	return &dockerSocketMountBenchmark{}
}
