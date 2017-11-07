package hostconfiguration

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type containerPartitionBenchmark struct{}

func (c *containerPartitionBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 1.1",
		Description:  "Ensure a separate partition for containers has been created",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *containerPartitionBenchmark) Run() (result common.TestResult) {
	fstab, err := common.ReadFile("/etc/fstab")
	if err != nil {
		result.Warn()
		return
	}
	if strings.Contains(fstab, "/var/lib/docker") {
		result.Pass()
		return
	}
	_, err = common.CombinedOutput("mountpoint", "-q", "--", "/var/lib/docker")
	if err == nil {
		result.Pass()
		return
	}
	result.Warn()
	result.AddNotes("/var/lib/docker does not have its own partition")
	return
}

// NewContainerPartitionBenchmark implements CIS-1.1
func NewContainerPartitionBenchmark() common.Benchmark {
	return &containerPartitionBenchmark{}
}
