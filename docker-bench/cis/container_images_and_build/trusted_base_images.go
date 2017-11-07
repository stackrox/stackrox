package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type trustedBaseImagesBenchmark struct{}

func (c *trustedBaseImagesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.2",
		Description:  "Ensure that containers use trusted base images",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *trustedBaseImagesBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Verification of trusted base images requires user specification")
	return
}

// NewTrustedBaseImagesBenchmark implements CIS-4.2
func NewTrustedBaseImagesBenchmark() common.Benchmark {
	return &trustedBaseImagesBenchmark{}
}
