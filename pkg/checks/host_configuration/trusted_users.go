package hostconfiguration

import (
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type trustedUsers struct{}

func (c *trustedUsers) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 1.4",
			Description: "Ensure only trusted users are allowed to control Docker daemon",
		},
	}
}

func (c *trustedUsers) Run() (result v1.CheckResult) {
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
