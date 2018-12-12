package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type imageSecretsBenchmark struct{}

func (c *imageSecretsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.10",
			Description: "Ensure secrets are not stored in Dockerfiles",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageSecretsBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Ensuring secrets are not stored in Dockerfiles requires manual introspection")
	return
}

// NewImageSecretsBenchmark implements CIS-4.10
func NewImageSecretsBenchmark() utils.Check {
	return &imageSecretsBenchmark{}
}
