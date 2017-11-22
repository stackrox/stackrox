package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type sshBenchmark struct{}

func (c *sshBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 5.6",
			Description: "Ensure ssh is not run within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *sshBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Check containers to ensure ssh is not running within them")
	return
}

// NewSSHBenchmark implements CIS-5.6
func NewSSHBenchmark() utils.Benchmark {
	return &sshBenchmark{}
}
