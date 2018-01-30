package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
	"github.com/opencontainers/runc/libcontainer/user"
)

type trustedUsers struct{}

func (c *trustedUsers) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 1.4",
			Description: "Ensure the container host has been Hardened",
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
