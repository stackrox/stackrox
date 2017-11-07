package containerruntime

import (
	//"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
	//"github.com/docker/docker/api/types"
)

type latestImageBenchmark struct{}

func (c *latestImageBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.27",
		Description:  "Ensure docker commands always get the latest version of the image",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *latestImageBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Pulling images is invasive and not always possible depending on credential management")
	return
}

// NewLatestImageBenchmark implements CIS-5.27
func NewLatestImageBenchmark() common.Benchmark {
	return &latestImageBenchmark{}
}
