package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type defaultUlimitBenchmark struct{}

func (c *defaultUlimitBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.7",
		Description:  "Ensure the default ulimit is configured appropriately",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *defaultUlimitBenchmark) Run() (result common.TestResult) {
	if _, ok := common.DockerConfig["default-ulimit"]; !ok {
		result.Warn()
		result.AddNotes("No default-ulimit values are set")
		return
	}
	result.Pass()
	return
}

// NewDefaultUlimitBenchmark implements CIS-2.7
func NewDefaultUlimitBenchmark() common.Benchmark {
	return &defaultUlimitBenchmark{}
}
