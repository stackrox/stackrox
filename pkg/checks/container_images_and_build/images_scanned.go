package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type scannedImagesBenchmark struct{}

func (c *scannedImagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.4",
			Description: "Ensure images are scanned and rebuilt to include security patches",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *scannedImagesBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Checking if images are scanned requires third party integration")
	return
}

// NewScannedImagesBenchmark implements CIS-4.4
func NewScannedImagesBenchmark() utils.Check {
	return &scannedImagesBenchmark{}
}
