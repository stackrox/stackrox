package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type imageSecretsBenchmark struct{}

func (c *imageSecretsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.10",
			Description: "Ensure secrets are not stored in Dockerfiles",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageSecretsBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Ensuring secrets are not stored in Dockerfiles requires manual introspection")
	return
}

// NewImageSecretsBenchmark implements CIS-4.10
func NewImageSecretsBenchmark() utils.Check {
	return &imageSecretsBenchmark{}
}
