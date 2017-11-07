package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
	"github.com/opencontainers/runc/libcontainer/user"
)

type trustedUsers struct{}

func (c *trustedUsers) Definition() common.Definition {
	return common.Definition{
		Name:        "CIS 1.4",
		Description: "Ensure the container host has been Hardened",
	}
}

func (c *trustedUsers) Run() (result common.TestResult) {
	group, err := user.LookupGroup("docker")
	if err != nil {
		result.Result = common.Warn
		result.AddNotef("Docker group does not exist: %v", err.Error())
		return
	}
	result.Result = common.Note
	result.AddNotes(group.List...)
	return
}

// NewTrustedUsers implements CIS-1.4
func NewTrustedUsers() common.Benchmark {
	return &trustedUsers{}
}
