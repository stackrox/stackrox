package dockerdaemonconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type daemonWideSeccompBenchmark struct{}

func (c *daemonWideSeccompBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.16",
			Description: "Ensure daemon-wide custom seccomp profile is applied, if needed",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *daemonWideSeccompBenchmark) Run() (result v1.BenchmarkCheckResult) {
	for _, opt := range utils.DockerInfo.SecurityOptions {
		if opt == "default" {
			utils.Warn(&result)
			utils.AddNotes(&result, "Default seccomp profile is enabled")
			return
		}
	}
	utils.Pass(&result)
	return
}

// NewDaemonWideSeccompBenchmark implements CIS-2.16
func NewDaemonWideSeccompBenchmark() utils.Check {
	return &daemonWideSeccompBenchmark{}
}
