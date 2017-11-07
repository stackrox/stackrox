package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type ipTablesEnabledBenchmark struct{}

func (c *ipTablesEnabledBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.3",
		Description:  "Ensure Docker is allowed to make changes to iptables",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *ipTablesEnabledBenchmark) Run() (result common.TestResult) {
	values, ok := common.DockerConfig["iptables"]
	if !ok {
		result.Pass()
		return
	}
	if values.Matches("false") {
		result.Warn()
		result.AddNotes("Docker is not configured to modify iptables")
		return
	}
	result.Pass()
	return
}

// NewIPTablesEnabledBenchmark implements CIS-2.3
func NewIPTablesEnabledBenchmark() common.Benchmark {
	return &ipTablesEnabledBenchmark{}
}
