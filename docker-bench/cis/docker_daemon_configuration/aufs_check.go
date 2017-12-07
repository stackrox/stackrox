package dockerdaemonconfiguration

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type aufsBenchmark struct{}

func (c *aufsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.5",
			Description: "Ensure aufs storage driver is not used",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *aufsBenchmark) Run() (result v1.CheckResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	if strings.Contains(info.Driver, "aufs") {
		utils.Warn(&result)
		utils.AddNotes(&result, "aufs is currently configured as the storage driver")
		return
	}
	utils.Pass(&result)
	return
}

// NewAUFSBenchmark implements CIS-2.5
func NewAUFSBenchmark() utils.Check {
	return &aufsBenchmark{}
}
