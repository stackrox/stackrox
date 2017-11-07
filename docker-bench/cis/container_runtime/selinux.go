package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type seLinuxBenchmark struct{}

func (c *seLinuxBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.2",
		Description:  "Ensure SELinux security options are set, if applicable",
		Dependencies: []common.Dependency{common.InitDockerConfig, common.InitContainers},
	}
}

func checkContainersForSELinux() (result common.TestResult) {
	result.Pass()
LOOP:
	for _, container := range common.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(strings.ToLower(opt), "selinux") {
				continue LOOP
			}
		}
		result.Warn()
		result.AddNotef("Container %v does not have selinux configured", container.ID)
	}
	return
}

func (c *seLinuxBenchmark) Run() (result common.TestResult) {
	if values, ok := common.DockerConfig["selinux-enabled"]; ok && (values.Matches("") || values.Matches("true")) {
		result = checkContainersForSELinux()
		return
	}
	result.Pass()
	return
}

// NewSELinuxBenchmark implements CIS-5.2
func NewSELinuxBenchmark() common.Benchmark {
	return &seLinuxBenchmark{}
}
