package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type authorizationPluginBenchmark struct{}

func (c *authorizationPluginBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.11",
		Description:  "Ensure that authorization for Docker client commands is enabled",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *authorizationPluginBenchmark) Run() (result common.TestResult) {
	_, ok := common.DockerConfig["authorization-plugin"]
	if !ok {
		result.Result = common.Warn
		result.AddNotes("No authorization plugin is enabled for the docker client")
		return
	}
	// TODO(cgorman) search for image?
	result.Result = common.Pass
	return
}

// NewAuthorizationPluginBenchmark implements CIS-2.11
func NewAuthorizationPluginBenchmark() common.Benchmark {
	return &authorizationPluginBenchmark{}
}
