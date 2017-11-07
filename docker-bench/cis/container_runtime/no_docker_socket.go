package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type dockerSocketMountBenchmark struct{}

func (c *dockerSocketMountBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.31",
		Description:  "Ensure the Docker socket is not mounted inside any containers",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *dockerSocketMountBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for _, containerMount := range container.Mounts {
			if strings.Contains(containerMount.Source, "docker.sock") {
				result.Warn()
				result.AddNotef("Container %v has mounted docker.sock", container.ID)
			}
		}
	}
	return
}

// NewDockerSocketMountBenchmark implements CIS-5.31
func NewDockerSocketMountBenchmark() common.Benchmark {
	return &dockerSocketMountBenchmark{}
}
