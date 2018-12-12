package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type trustedBaseImagesBenchmark struct{}

func (c *trustedBaseImagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.2",
			Description: "Ensure that containers use trusted base images",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *trustedBaseImagesBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Verification of trusted base images requires user specification")
	return
}

// NewTrustedBaseImagesBenchmark implements CIS-4.2
func NewTrustedBaseImagesBenchmark() utils.Check {
	return &trustedBaseImagesBenchmark{}
}
