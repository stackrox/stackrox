package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type sensitiveHostMountsBenchmark struct{}

func (c *sensitiveHostMountsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
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

func (c *sensitiveHostMountsBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, container := range utils.ContainersRunning {
		for _, mount := range container.Mounts {
			if _, ok := sensitiveMountMap[mount.Source]; ok {
				utils.Warn(&result)
				utils.AddNotef(&result, "Container '%v' mounts in sensitive mount source '%v'", container.ID, mount.Source)
			}
		}
	}
	return
}

// NewSensitiveHostMountsBenchmark implements CIS-5.5
func NewSensitiveHostMountsBenchmark() utils.Check {
	return &sensitiveHostMountsBenchmark{}
}
