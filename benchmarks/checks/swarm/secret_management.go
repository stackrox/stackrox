package swarm

import (
	"github.com/docker/docker/api/types"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
)

type secretManagement struct{}

func (c *secretManagement) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.5",
			Description: "Ensure Docker's secret management commands are used for managing secrets in a Swarm cluster",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *secretManagement) Run() (result v1.CheckResult) {
	if !utils.DockerInfo.Swarm.ControlAvailable {
		utils.NotApplicable(&result)
		utils.AddNotes(&result, "Checking docker secrets applies only to Swarm managers and this node is not a Swarm Manager")
		return
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	secretsList, err := utils.DockerClient.SecretList(ctx, types.SecretListOptions{})
	if err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Unable to get secrets: %+v", err)
		return
	}
	if len(secretsList) == 0 {
		utils.Warn(&result)
		utils.AddNotes(&result, "Docker secrets is not being utilized")
		return
	}
	utils.Pass(&result)
	utils.AddNotef(&result, "Currently %v Docker secrets are being utilized", len(secretsList))
	return
}

// NewSecretManagement implements CIS-7.5
func NewSecretManagement() utils.Check {
	return &secretManagement{}
}
