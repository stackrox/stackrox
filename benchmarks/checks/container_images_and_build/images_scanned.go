package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type scannedImagesBenchmark struct{}

func (c *scannedImagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.4",
			Description: "Ensure images are scanned and rebuilt to include security patches",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *scannedImagesBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Checking if images are scanned requires third party integration")
	return
}

// NewScannedImagesBenchmark implements CIS-4.4
func NewScannedImagesBenchmark() utils.Check {
	return &scannedImagesBenchmark{}
}
