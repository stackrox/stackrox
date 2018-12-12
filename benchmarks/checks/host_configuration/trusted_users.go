package hostconfiguration

import (
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type trustedUsers struct{}

func (c *trustedUsers) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 1.4",
			Description: "Ensure only trusted users are allowed to control Docker daemon",
		},
	}
}

func (c *trustedUsers) Run() (result storage.BenchmarkCheckResult) {
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
func NewTrustedUsers() utils.Check {
	return &trustedUsers{}
}
