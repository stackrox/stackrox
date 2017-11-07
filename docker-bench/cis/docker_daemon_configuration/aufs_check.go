package dockerdaemonconfiguration

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type aufsBenchmark struct{}

func (c *aufsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.5",
		Description:  "Ensure aufs storage driver is not used",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *aufsBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Result = common.Warn
		result.AddNotes(err.Error())
		return
	}
	if strings.Contains(info.Driver, "aufs") {
		result.Result = common.Warn
		result.AddNotes("aufs is currently configured as the storage driver")
		return
	}
	result.Result = common.Pass
	return
}

// NewAUFSBenchmark implements CIS-2.5
func NewAUFSBenchmark() common.Benchmark {
	return &aufsBenchmark{}
}
