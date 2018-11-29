package swarm

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type hostInterfaceBind struct{}

func (c *hostInterfaceBind) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.3",
			Description: "Ensure swarm services are binded to a specific host interface",
		},
		Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *hostInterfaceBind) Run() (result v1.BenchmarkCheckResult) {
	_, exists := utils.DockerConfig.Get("swarm-default-advertise-addr")
	if !exists {
		utils.Warn(&result)
		utils.AddNotef(&result, "swarm-default-advertise-addr is not specified and it defaults to 0.0.0.0:2377 which binds to all interfaces")
		return
	}
	utils.Pass(&result)
	return
}

// NewHostInterfaceBind implements CIS-7.1
func NewHostInterfaceBind() utils.Check {
	return &hostInterfaceBind{}
}
