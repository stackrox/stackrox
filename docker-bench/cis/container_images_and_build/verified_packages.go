package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type verifiedPackagesBenchmark struct{}

func (c *verifiedPackagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 4.11",
			Description: "Ensure verified packages are only Installed",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *verifiedPackagesBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Note(&result)
	utils.AddNotef(&result, "Checking if verified packages are only installed requires manual introspection")
	return
}

// NewVerifiedPackagesBenchmark implements CIS-4.11
func NewVerifiedPackagesBenchmark() utils.Benchmark {
	return &verifiedPackagesBenchmark{}
}
