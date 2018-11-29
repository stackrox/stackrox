package swarm

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type autoLockRotation struct{}

func (c *autoLockRotation) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.7",
			Description: "Ensure swarm manager auto-lock key is rotated periodically",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *autoLockRotation) Run() (result v1.BenchmarkCheckResult) {
	if !utils.DockerInfo.Swarm.ControlAvailable {
		utils.NotApplicable(&result)
		utils.AddNotes(&result, "Autolock applies only to Swarm managers and this node is not a Swarm Manager")
		return
	}
	utils.Note(&result)
	utils.AddNotes(&result, "Rotate the auto-lock key periodically using 'docker swarm unlock-key --rotate'")
	return
}

// NewAutoLockRotationCheck implements CIS-7.7
func NewAutoLockRotationCheck() utils.Check {
	return &autoLockRotation{}
}
