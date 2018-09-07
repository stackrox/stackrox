package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type tlsVerifyBenchmark struct{}

func (c *tlsVerifyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.6",
			Description: "Ensure TLS authentication for Docker daemon is configured",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *tlsVerifyBenchmark) Run() (result v1.CheckResult) {
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
func NewTLSVerifyBenchmark() utils.Check {
	return &tlsVerifyBenchmark{}
}
