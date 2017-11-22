package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type tlsVerifyBenchmark struct{}

func (c *tlsVerifyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.6",
			Description: "Ensure TLS authentication for Docker daemon is configured",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *tlsVerifyBenchmark) Run() (result v1.BenchmarkTestResult) {
	hosts, ok := utils.DockerConfig["host"]
	if !ok {
		utils.Pass(&result)
		utils.AddNotes(&result, "Docker doesn't expose the docker socket over tcp")
		return
	}
	if _, exists := hosts.Contains("fd://"); exists {
		utils.Pass(&result)
		utils.AddNotes(&result, "Docker doesn't expose the docker socket over tcp")
		return
	}
	// Check TLS
	if _, ok := utils.DockerConfig["tlsverify"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "tlsverify is not set")
		return
	}
	if _, ok := utils.DockerConfig["tlscacert"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "tlscacert is not set")
		return
	}
	if _, ok := utils.DockerConfig["tlscert"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "tlscert is not set")
		return
	}

	if _, ok := utils.DockerConfig["tlskey"]; !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "tlskey is not set")
		return
	}
	utils.Pass(&result)
	return
}

// NewTLSVerifyBenchmark implements CIS-2.6
func NewTLSVerifyBenchmark() utils.Benchmark {
	return &tlsVerifyBenchmark{}
}
