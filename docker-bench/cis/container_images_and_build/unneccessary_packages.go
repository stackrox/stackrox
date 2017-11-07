package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type unnecessaryPackagesBenchmark struct{}

func (c *unnecessaryPackagesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.3",
		Description:  "Ensure unnecessary packages are not installed in the container",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *unnecessaryPackagesBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotef("Checking if packages inside the image are necessary container ar and make sure they are necessary")
	return
}

// NewUnnecessaryPackagesBenchmark implements CIS-4.3
func NewUnnecessaryPackagesBenchmark() common.Benchmark {
	return &unnecessaryPackagesBenchmark{}
}
