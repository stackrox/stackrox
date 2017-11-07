package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type userDockerExecBenchmark struct{}

func (c *userDockerExecBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.23",
		Description:  "Ensure docker exec commands are not used with user option",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *userDockerExecBenchmark) Run() (result common.TestResult) {
	result.Pass()
	auditLog, err := common.ReadFile("/var/log/audit/audit.log")
	if err != nil {
		result.Warn()
		result.AddNotef("Error reading /var/log/audit/audit.log: %+v", err)
		return
	}
	lines := strings.Split(auditLog, "\n")
	for _, line := range lines {
		if strings.Contains(line, "exec") && strings.Contains(line, "user") {
			result.Warn()
			result.AddNotef("docker exec was used with the --user option: %v", line)
		}
	}
	return
}

// NewUserDockerExecBenchmark implements CIS-5.23
func NewUserDockerExecBenchmark() common.Benchmark {
	return &userDockerExecBenchmark{}
}
