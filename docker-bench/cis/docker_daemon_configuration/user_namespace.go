package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type userNamespaceBenchmark struct{}

func (c *userNamespaceBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.8",
		Description:  "Enable user namespace support",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *userNamespaceBenchmark) Run() (result common.TestResult) {
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Result = common.Warn
		result.AddNotes(err.Error())
		return
	}
	for _, opt := range info.SecurityOptions {
		if opt == "userns" {
			result.Result = common.Pass
			return
		}
	}
	result.Result = common.Warn
	result.AddNotes("userns is not present in security options")
	return
}

// NewUserNamespaceBenchmark implements CIS-2.8
func NewUserNamespaceBenchmark() common.Benchmark {
	return &userNamespaceBenchmark{}
}
