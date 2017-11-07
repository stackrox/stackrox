package dockerdaemonconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
	"github.com/docker/docker/api/types/filters"
)

type networkRestrictionBenchmark struct{}

func (c *networkRestrictionBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.1",
		Description:  "Ensure network traffic is restricted between containers on the default bridge",
		Dependencies: []common.Dependency{common.InitDockerConfig, common.InitDockerClient},
	}
}

func (c *networkRestrictionBenchmark) Run() (result common.TestResult) {
	listFilters := filters.NewArgs()
	listFilters.Add("Name", "bridge")
	inspect, err := common.DockerClient.NetworkInspect(context.Background(), "bridge")
	if err != nil {
		result.Result = common.Warn
		result.AddNotes(err.Error())
		return
	}
	if inspect.Options["com.docker.network.bridge.enable_icc"] == "true" {
		result.Result = common.Warn
		result.AddNotes("Enable icc is true on bridge network")
		return
	}
	result.Result = common.Pass
	return
}

// NewNetworkRestrictionBenchmark implements CIS-2.1
func NewNetworkRestrictionBenchmark() common.Benchmark {
	return &networkRestrictionBenchmark{}
}
