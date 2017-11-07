package containerruntime

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type privilegedDockerExecBenchmark struct{}

func (c *privilegedDockerExecBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 5.22",
		Description:  "Ensure docker exec commands are not used with privileged option",
		Dependencies: []common.Dependency{common.InitContainers},
	}
}

func (c *privilegedDockerExecBenchmark) Run() (result common.TestResult) {
	result.Pass()
	auditLog, err := common.ReadFile("/var/log/audit/audit.log")
	if err != nil {
		result.Warn()
		result.AddNotef("Error reading /var/log/audit/audit.log: %+v", err)
		return
	}
	lines := strings.Split(auditLog, "\n")
	for _, line := range lines {
		if strings.Contains(line, "exec") && strings.Contains(line, "privileged") {
			result.Warn()
			result.AddNotef("docker exec was used with the --privileged option: %v", line)
		}
	}
	return
}

// NewPrivilegedDockerExecBenchmark implements CIS-5.22
func NewPrivilegedDockerExecBenchmark() common.Benchmark {
	return &privilegedDockerExecBenchmark{}
}
