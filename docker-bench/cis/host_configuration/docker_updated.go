package hostconfiguration

import (
	"context"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type dockerUpdated struct{}

func (c *dockerUpdated) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 1.3",
		Description:  "Ensure Docker is up to date",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *dockerUpdated) Run() (result common.TestResult) {
	version, err := common.DockerClient.ServerVersion(context.Background())
	if err != nil {
		result.Note()
		result.AddNotef("Manual introspection will be req'd for docker version. Could not retrieve due to %+v", err)
		return
	}
	result.Result = common.Note
	result.AddNotes("Docker server is currently running %v", version.Version)
	return
}

// NewDockerUpdated implements CIS-1.3
func NewDockerUpdated() common.Benchmark {
	return &dockerUpdated{}
}
