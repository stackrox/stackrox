package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type pidCgroupBenchmark struct{}

func (c *pidCgroupBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.28",
			Description: "Ensure PIDs cgroup limit is used",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *pidCgroupBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.PidsLimit <= 0 {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' does not have pids limit set", container.ID)
		}
	}
	return
}

// NewPidCgroupBenchmark implements CIS-5.28
func NewPidCgroupBenchmark() utils.Check {
	return &pidCgroupBenchmark{}
}
