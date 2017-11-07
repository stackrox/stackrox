package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type hostHardened struct{}

func (c *hostHardened) Definition() common.Definition {
	return common.Definition{
		Name:        "CIS 1.2",
		Description: "Ensure the container host has been Hardened",
	}
}

func (c *hostHardened) Run() (result common.TestResult) {
	result.Result = common.Note
	result.AddNotes("Ensuring the host is hardened with the lastest kernel requires manual introspection")
	return
}

// NewHostHardened implements CIS-1.2
func NewHostHardened() common.Benchmark {
	return &hostHardened{}
}
