package swarm

import (
	"github.com/docker/docker/api/types/swarm"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type swarmEnabled struct{}

func (c *swarmEnabled) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.1",
			Description: "Do not enable swarm mode on a docker engine instance unless needed",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *swarmEnabled) Run() (result storage.BenchmarkCheckResult) {
	if utils.DockerInfo.Swarm.LocalNodeState != swarm.LocalNodeStateActive && utils.DockerInfo.Swarm.LocalNodeState != swarm.LocalNodeStateInactive {
		utils.Warn(&result)
		utils.AddNotef(&result, "Node is in unexpected state: %v", utils.DockerInfo.Swarm.LocalNodeState)
		return
	}
	utils.Note(&result)
	utils.AddNotef(&result, "Node swarm state is currently %v", utils.DockerInfo.Swarm.LocalNodeState)
	return
}

// NewSwarmEnabledCheck implements CIS-7.1
func NewSwarmEnabledCheck() utils.Check {
	return &swarmEnabled{}
}
