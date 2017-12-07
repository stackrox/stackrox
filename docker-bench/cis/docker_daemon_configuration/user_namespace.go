package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type userNamespaceBenchmark struct{}

func (c *userNamespaceBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.8",
			Description: "Enable user namespace support",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *userNamespaceBenchmark) Run() (result v1.CheckResult) {
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	for _, opt := range info.SecurityOptions {
		if opt == "userns" {
			utils.Pass(&result)
			return
		}
	}
	utils.Warn(&result)
	utils.AddNotes(&result, "userns is not present in security options")
	return
}

// NewUserNamespaceBenchmark implements CIS-2.8
func NewUserNamespaceBenchmark() utils.Check {
	return &userNamespaceBenchmark{}
}
