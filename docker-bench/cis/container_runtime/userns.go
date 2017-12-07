package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type usernsBenchmark struct{}

func (c *usernsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.30",
			Description: "Ensure the host's user namespaces is not shared",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *usernsBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.HostConfig.UsernsMode.IsHost() {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container %v has user namespace set to host", container.ID)
		}
	}
	return
}

// NewUsernsBenchmark implements CIS-5.30
func NewUsernsBenchmark() utils.Check {
	return &usernsBenchmark{}
}
