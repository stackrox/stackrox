package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type imageSecretsBenchmark struct{}

func (c *imageSecretsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.10",
		Description:  "Ensure secrets are not stored in Dockerfiles",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *imageSecretsBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Ensuring secrets are not stored in Dockerfiles requires manual introspection")
	return
}

// NewImageSecretsBenchmark implements CIS-4.10
func NewImageSecretsBenchmark() common.Benchmark {
	return &imageSecretsBenchmark{}
}
