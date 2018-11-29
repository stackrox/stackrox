package swarm

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type mgmtPlaneData struct{}

func (c *mgmtPlaneData) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.10",
			Description: "Ensure management plane traffic has been separated from data plane traffic",
		},
	}
}

func (c *mgmtPlaneData) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotef(&result, "Check each swarm node and ensure that the data plane traffic and management plane traffic are segmented")
	return
}

// NewManagementPlaneCheck implements CIS-7.10
func NewManagementPlaneCheck() utils.Check {
	return &mgmtPlaneData{}
}
