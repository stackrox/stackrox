package containerimagesandbuild

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type trustedBaseImagesBenchmark struct{}

func (c *trustedBaseImagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.2",
			Description: "Ensure that containers use trusted base images",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *trustedBaseImagesBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Verification of trusted base images requires user specification")
	return
}

// NewTrustedBaseImagesBenchmark implements CIS-4.2
func NewTrustedBaseImagesBenchmark() utils.Check {
	return &trustedBaseImagesBenchmark{}
}
