package containerimagesandbuild

// Ensure Content trust for Docker is Enabled

import (
	"os"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type contentTrustBenchmark struct{}

func (c *contentTrustBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.5",
			Description: "Ensure Content trust for Docker is Enabled",
		},
	}
}

func (c *contentTrustBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	trust := os.Getenv("DOCKER_CONTENT_TRUST")
	if trust == "" {
		utils.Warn(&result)
		utils.AddNotes(&result, "DOCKER_CONTENT_TRUST defaults to 0 and it is unset")
		return
	}
	if trust != "1" {
		utils.Warn(&result)
		utils.AddNotef(&result, "DOCKER_CONTENT_TRUST is set to %v", trust)
		return
	}
	return
}

// NewContentTrustBenchmark implements CIS-4.5
func NewContentTrustBenchmark() utils.Check {
	return &contentTrustBenchmark{}
}
