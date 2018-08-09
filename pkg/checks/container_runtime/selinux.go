package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type seLinuxBenchmark struct{}

func (c *seLinuxBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.2",
			Description: "Ensure SELinux security options are set, if applicable",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig, utils.InitContainers},
	}
}

func checkContainersForSELinux() (result v1.CheckResult) {
	utils.Pass(&result)
LOOP:
	for _, container := range utils.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(strings.ToLower(opt), "selinux") {
				continue LOOP
			}
		}
		utils.Warn(&result)
		utils.AddNotef(&result, "Container '%v' (%v) does not have selinux configured", container.ID, container.Name)
	}
	return
}

func (c *seLinuxBenchmark) Run() (result v1.CheckResult) {
	if values, ok := utils.DockerConfig["selinux-enabled"]; ok && (values.Matches("") || values.Matches("true")) {
		result = checkContainersForSELinux()
		return
	}
	utils.Pass(&result)
	return
}

// NewSELinuxBenchmark implements CIS-5.2
func NewSELinuxBenchmark() utils.Check {
	return &seLinuxBenchmark{}
}
