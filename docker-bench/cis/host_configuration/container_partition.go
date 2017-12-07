package hostconfiguration

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type containerPartitionBenchmark struct{}

func (c *containerPartitionBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 1.1",
			Description: "Ensure a separate partition for containers has been created",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *containerPartitionBenchmark) Run() (result v1.CheckResult) {
	fstab, err := utils.ReadFile("/etc/fstab")
	if err != nil {
		utils.Warn(&result)
		return
	}
	if strings.Contains(fstab, "/var/lib/docker") {
		utils.Pass(&result)
		return
	}
	_, err = utils.CombinedOutput("mountpoint", "-q", "--", "/var/lib/docker")
	if err == nil {
		utils.Pass(&result)
		return
	}
	utils.Warn(&result)
	utils.AddNotes(&result, "/var/lib/docker does not have its own partition")
	return
}

// NewContainerPartitionBenchmark implements CIS-1.1
func NewContainerPartitionBenchmark() utils.Check {
	return &containerPartitionBenchmark{}
}
