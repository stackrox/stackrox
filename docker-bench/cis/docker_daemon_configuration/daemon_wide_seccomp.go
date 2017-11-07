package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type daemonWideSeccompBenchmark struct{}

func (c *daemonWideSeccompBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.16",
		Description:  "Ensure daemon-wide custom seccomp profile is applied, if needed",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *daemonWideSeccompBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Warn()
		result.AddNotes(err.Error())
		return
	}
	for _, opt := range info.SecurityOptions {
		if opt == "default" {
			result.Warn()
			result.AddNotes("Default seccomp profile is enabled")
			return
		}
	}
	result.Pass()
	return
}

// NewDaemonWideSeccompBenchmark implements CIS-2.16
func NewDaemonWideSeccompBenchmark() common.Benchmark {
	return &daemonWideSeccompBenchmark{}
}
