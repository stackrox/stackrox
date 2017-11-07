package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type verifiedPackagesBenchmark struct{}

func (c *verifiedPackagesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.11",
		Description:  "Ensure verified packages are only Installed",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *verifiedPackagesBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotef("Checking if verified packages are only installed requires manual introspection")
	return
}

// NewVerifiedPackagesBenchmark implements CIS-4.11
func NewVerifiedPackagesBenchmark() common.Benchmark {
	return &verifiedPackagesBenchmark{}
}
