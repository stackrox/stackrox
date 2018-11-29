package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type setuidSetGidPermissionsBenchmark struct{}

func (c *setuidSetGidPermissionsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.8",
			Description: "Ensure setuid and setgid permissions are removed in the images",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *setuidSetGidPermissionsBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Checking if setuid and setgid permissions are removed in the images is invasive and requires running every image")
	return
}

// NewSetuidSetGidPermissionsBenchmark implements CIS-4.8
func NewSetuidSetGidPermissionsBenchmark() utils.Check {
	return &setuidSetGidPermissionsBenchmark{}
}
