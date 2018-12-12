package swarm

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type autoLock struct{}

func (c *autoLock) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.6",
			Description: "Ensure swarm manager is run in auto-lock mode",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *autoLock) Run() (result storage.BenchmarkCheckResult) {
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
