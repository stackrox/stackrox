package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type sshBenchmark struct{}

func (c *sshBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.6",
			Description: "Ensure ssh is not run within containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *sshBenchmark) Run() (result v1.CheckResult) {
	utils.Note(&result)
	utils.AddNotes(&result, "Check containers to ensure ssh is not running within them")
	return
}

// NewSSHBenchmark implements CIS-5.6
func NewSSHBenchmark() utils.Check {
	return &sshBenchmark{}
}
