package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type setuidSetGidPermissionsBenchmark struct{}

func (c *setuidSetGidPermissionsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.8",
		Description:  "Ensure setuid and setgid permissions are removed in the images",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *setuidSetGidPermissionsBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Checking if setuid and setgid permissions are removed in the images is invasive and requires running every image")
	return
}

// NewSetuidSetGidPermissionsBenchmark implements CIS-4.8
func NewSetuidSetGidPermissionsBenchmark() common.Benchmark {
	return &setuidSetGidPermissionsBenchmark{}
}
