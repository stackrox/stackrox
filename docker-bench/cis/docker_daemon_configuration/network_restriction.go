package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

type networkRestrictionBenchmark struct{}

func (c *networkRestrictionBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.1",
			Description: "Ensure network traffic is restricted between containers on the default bridge",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig, utils.InitDockerClient},
	}
}

func (c *networkRestrictionBenchmark) Run() (result v1.CheckResult) {
	listFilters := filters.NewArgs()
	listFilters.Add("Name", "bridge")
	inspect, err := utils.DockerClient.NetworkInspect(context.Background(), "bridge", types.NetworkInspectOptions{})
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	if inspect.Options["com.docker.network.bridge.enable_icc"] == "true" {
		utils.Warn(&result)
		utils.AddNotes(&result, "Enable icc is true on bridge network")
		return
	}
	utils.Pass(&result)
	return
}

// NewNetworkRestrictionBenchmark implements CIS-2.1
func NewNetworkRestrictionBenchmark() utils.Check {
	return &networkRestrictionBenchmark{}
}
