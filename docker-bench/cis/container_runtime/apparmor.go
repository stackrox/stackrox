package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type appArmorBenchmark struct{}

func (c *appArmorBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.1",
		Description:  "Ensure AppArmor Profile is Enabled",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *appArmorBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		if container.AppArmorProfile == "" || container.AppArmorProfile == "unconfined" {
			result.Warn()
			result.AddNotef("Container %v does not have app armor configured", container.ID)
		}
	}
	return
}

// NewAppArmorBenchmark implements CIS-5.1
func NewAppArmorBenchmark() common.Benchmark {
	return &appArmorBenchmark{}
}
