package containerimagesandbuild

// Ensure Content trust for Docker is Enabled

import (
	"os"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type contentTrustBenchmark struct{}

func (c *contentTrustBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:        "CIS 4.5",
		Description: "Ensure Content trust for Docker is Enabled",
	}
}

func (c *contentTrustBenchmark) Run() (result common.TestResult) {
	result.Pass()
	trust := os.Getenv("DOCKER_CONTENT_TRUST")
	if trust == "" {
		result.Warn()
		result.AddNotes("DOCKER_CONTENT_TRUST defaults to 0 and it is unset")
		return
	}
	if trust != "1" {
		result.Warn()
		result.AddNotef("DOCKER_CONTENT_TRUST is set to %v", trust)
		return
	}
	return
}

// NewContentTrustBenchmark implements CIS-4.5
func NewContentTrustBenchmark() common.Benchmark {
	return &contentTrustBenchmark{}
}
