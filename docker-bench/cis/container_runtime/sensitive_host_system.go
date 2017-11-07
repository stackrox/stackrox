package containerruntime

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type sensitiveHostMountsBenchmark struct{}

func (c *sensitiveHostMountsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.5",
		Description:  "Ensure sensitive host system directories are not mounted on containers",
		Dependencies: []common.Dependency{common.InitContainers},
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

func (c *sensitiveHostMountsBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, container := range common.ContainersRunning {
		for _, mount := range container.Mounts {
			if _, ok := sensitiveMountMap[mount.Source]; ok {
				result.Warn()
				result.AddNotef("Container %v mounts in sensitive mount source %v", container.ID, mount.Source)
			}
		}
	}
	return
}

// NewSensitiveHostMountsBenchmark implements CIS-5.5
func NewSensitiveHostMountsBenchmark() common.Benchmark {
	return &sensitiveHostMountsBenchmark{}
}
