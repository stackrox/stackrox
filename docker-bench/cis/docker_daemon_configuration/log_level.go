package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type logLevelBenchmark struct{}

func (c *logLevelBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.2",
		Description:  "Ensure the logging level is set to 'info'",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *logLevelBenchmark) Run() (result common.TestResult) {
	if vals, ok := common.DockerConfig["log-level"]; ok {
		if _, exists := vals.Contains("info"); !exists {
			result.Result = common.Warn
			result.AddNotef("log-level is set to %v", vals[0])
			return
		}
	}
	result.Result = common.Pass
	return
}

// NewLogLevelBenchmark implements CIS-2.2
func NewLogLevelBenchmark() common.Benchmark {
	return &logLevelBenchmark{}
}
