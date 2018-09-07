package hostconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type hostHardened struct{}

func (c *hostHardened) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 1.2",
			Description: "Ensure the container host has been Hardened",
		},
	}
}

func (c *hostHardened) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Ensuring the host is hardened with the latest kernel requires manual introspection")
	return
}

// NewHostHardened implements CIS-1.2
func NewHostHardened() utils.Check {
	return &hostHardened{}
}
