package swarm

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type nodeCertificates struct{}

func (c *nodeCertificates) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.8",
			Description: "Ensure node certificates are rotated as appropriate",
		},
		Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *nodeCertificates) Run() (result v1.BenchmarkCheckResult) {
	if !utils.DockerInfo.Swarm.ControlAvailable {
		utils.NotApplicable(&result)
		utils.AddNotes(&result, "Checking  node certificate rotation applies only to Swarm managers and this node is not a Swarm Manager")
		return
	}
	utils.Note(&result)
	days := int(utils.DockerInfo.Swarm.Cluster.Spec.CAConfig.NodeCertExpiry.Hours() / 24)
	utils.AddNotef(&result, "Check that the node certificates are rotated periodically. Expiry time is currently set to %v days", days)
	return
}

// NewNodeCertificates implements CIS-7.8
func NewNodeCertificates() utils.Check {
	return &nodeCertificates{}
}
