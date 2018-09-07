package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type unnecessaryPackagesBenchmark struct{}

func (c *unnecessaryPackagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.3",
			Description: "Ensure unnecessary packages are not installed in the container",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *unnecessaryPackagesBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotef(&result, "Check if the packages inside the image are necessary")
	return
}

// NewUnnecessaryPackagesBenchmark implements CIS-4.3
func NewUnnecessaryPackagesBenchmark() utils.Check {
	return &unnecessaryPackagesBenchmark{}
}
