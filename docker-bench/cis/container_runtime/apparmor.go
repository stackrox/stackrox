package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type appArmorBenchmark struct{}

func (c *appArmorBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 5.1",
			Description: "Ensure AppArmor Profile is Enabled",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *appArmorBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		if container.AppArmorProfile == "" || container.AppArmorProfile == "unconfined" {
			utils.Warn(&result)
			utils.AddNotef(&result, "Container '%v' does not have app armor configured", container.ID)
		}
	}
	return
}

// NewAppArmorBenchmark implements CIS-5.1
func NewAppArmorBenchmark() utils.Check {
	return &appArmorBenchmark{}
}
