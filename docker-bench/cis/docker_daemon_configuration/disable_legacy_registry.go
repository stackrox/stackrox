package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type disableLegacyRegistryBenchmark struct{}

func (c *disableLegacyRegistryBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.13",
		Description:  "Ensure operations on legacy registry (v1) are Disabled",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *disableLegacyRegistryBenchmark) Run() (result common.TestResult) {
	if _, ok := common.DockerConfig["disable-legacy-registry"]; !ok {
		result.Warn()
		result.AddNotes("Legacy registry is not disabled")
		return
	}
	result.Pass()
	return
}

// NewDisableLegacyRegistryBenchmark implements CIS-2.13
func NewDisableLegacyRegistryBenchmark() common.Benchmark {
	return &disableLegacyRegistryBenchmark{}
}
