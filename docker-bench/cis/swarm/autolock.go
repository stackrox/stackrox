package swarm

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type autoLock struct{}

func (c *autoLock) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 7.6",
			Description: "Ensure swarm manager is run in auto-lock mode",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *autoLock) Run() (result v1.CheckResult) {
	if !utils.DockerInfo.Swarm.ControlAvailable {
		utils.NotApplicable(&result)
		utils.AddNotes(&result, "Autolock applies only to Swarm managers and this node is not a Swarm Manager")
		return
	}
	if utils.DockerInfo.Swarm.Cluster.Spec.EncryptionConfig.AutoLockManagers {
		utils.Pass(&result)
		return
	}
	utils.Warn(&result)
	utils.AddNotes(&result, "Autolock is not configured on your swarm cluster")
	return
}

// NewAutoLockCheck implements CIS-7.6
func NewAutoLockCheck() utils.Check {
	return &autoLock{}
}
