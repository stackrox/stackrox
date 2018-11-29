package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type sshBenchmark struct{}

func (c *sshBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.6",
			Description: "Ensure ssh is not run within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *sshBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Check containers to ensure ssh is not running within them")
	return
}

// NewSSHBenchmark implements CIS-5.6
func NewSSHBenchmark() utils.Check {
	return &sshBenchmark{}
}
