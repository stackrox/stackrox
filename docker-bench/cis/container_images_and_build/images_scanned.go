package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type scannedImagesBenchmark struct{}

func (c *scannedImagesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.4",
		Description:  "Ensure images are scanned and rebuilt to include security patches",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *scannedImagesBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Checking if images are scanned requires third party integration")
	return
}

// NewScannedImagesBenchmark implements CIS-4.4
func NewScannedImagesBenchmark() common.Benchmark {
	return &scannedImagesBenchmark{}
}
