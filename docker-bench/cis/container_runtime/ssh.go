package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type sshBenchmark struct{}

func (c *sshBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.6",
		Description:  "Ensure ssh is not run within containers",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *sshBenchmark) Run() (result common.TestResult) {
	result.Note()
	result.AddNotes("Check containers to ensure ssh is not running within them")
	return
}

// NewSSHBenchmark implements CIS-5.6
func NewSSHBenchmark() common.Benchmark {
	return &sshBenchmark{}
}
