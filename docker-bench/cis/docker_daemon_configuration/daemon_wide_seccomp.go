package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type daemonWideSeccompBenchmark struct{}

func (c *daemonWideSeccompBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.16",
			Description: "Ensure daemon-wide custom seccomp profile is applied, if needed",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *daemonWideSeccompBenchmark) Run() (result v1.CheckResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	for _, opt := range info.SecurityOptions {
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
