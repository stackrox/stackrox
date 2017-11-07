package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type tlsVerifyBenchmark struct{}

func (c *tlsVerifyBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.6",
		Description:  "Ensure TLS authentication for Docker daemon is configured",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *tlsVerifyBenchmark) Run() (result common.TestResult) {
	hosts, ok := common.DockerConfig["host"]
	if !ok {
		result.Result = common.Pass
		result.AddNotes("Docker doesn't expose the docker socket over tcp")
		return
	}
	if _, exists := hosts.Contains("fd://"); exists {
		result.Result = common.Pass
		result.AddNotes("Docker doesn't expose the docker socket over tcp")
		return
	}
	// Check TLS
	if _, ok := common.DockerConfig["tlsverify"]; !ok {
		result.Result = common.Warn
		result.AddNotes("tlsverify is not set")
		return
	}
	if _, ok := common.DockerConfig["tlscacert"]; !ok {
		result.Result = common.Warn
		result.AddNotes("tlscacert is not set")
		return
	}
	if _, ok := common.DockerConfig["tlscert"]; !ok {
		result.Result = common.Warn
		result.AddNotes("tlscert is not set")
		return
	}

	if _, ok := common.DockerConfig["tlskey"]; !ok {
		result.Result = common.Warn
		result.AddNotes("tlskey is not set")
		return
	}
	result.Result = common.Pass
	return
}

// NewTLSVerifyBenchmark implements CIS-2.6
func NewTLSVerifyBenchmark() common.Benchmark {
	return &tlsVerifyBenchmark{}
}
