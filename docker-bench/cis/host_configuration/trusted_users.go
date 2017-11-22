package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/opencontainers/runc/libcontainer/user"
)

type trustedUsers struct{}

func (c *trustedUsers) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 1.4",
			Description: "Ensure the container host has been Hardened",
		},
	}
}

func (c *trustedUsers) Run() (result v1.BenchmarkTestResult) {
	group, err := user.LookupGroup("docker")
	if err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Docker group does not exist: %v", err.Error())
		return
	}
	utils.Note(&result)
	utils.AddNotes(&result, group.List...)
	return
}

// NewTrustedUsers implements CIS-1.4
func NewTrustedUsers() utils.Benchmark {
	return &trustedUsers{}
}
