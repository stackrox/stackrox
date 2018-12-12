package containerruntime

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type sensitiveHostMountsBenchmark struct{}

func (c *sensitiveHostMountsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.5",
			Description: "Ensure sensitive host system directories are not mounted on containers",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

var sensitiveMountMap = map[string]struct{}{
	"/":     {},
	"/boot": {},
	"/dev":  {},
	"/etc":  {},
	"/lib":  {},
	"/proc": {},
	"/sys":  {},
	"/usr":  {},
}

func (c *sensitiveHostMountsBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, mount := range container.Mounts {
			if _, ok := sensitiveMountMap[mount.Source]; ok {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container '%v' (%v) mounts in sensitive mount source '%v'", container.ID, container.Name, mount.Source)
			}
		}
	}
	return
}

// NewSensitiveHostMountsBenchmark implements CIS-5.5
func NewSensitiveHostMountsBenchmark() utils.Check {
	return &sensitiveHostMountsBenchmark{}
}
