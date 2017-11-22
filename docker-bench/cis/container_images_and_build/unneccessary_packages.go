package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type unnecessaryPackagesBenchmark struct{}

func (c *unnecessaryPackagesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 4.3",
			Description: "Ensure unnecessary packages are not installed in the container",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *unnecessaryPackagesBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Note(&result)
	utils.AddNotef(&result, "Checking if packages inside the image are necessary container ar and make sure they are necessary")
	return
}

// NewUnnecessaryPackagesBenchmark implements CIS-4.3
func NewUnnecessaryPackagesBenchmark() utils.Benchmark {
	return &unnecessaryPackagesBenchmark{}
}
