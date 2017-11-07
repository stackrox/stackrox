package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type disableUserlandProxyBenchmark struct{}

func (c *disableUserlandProxyBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.15",
		Description:  "Ensure Userland Proxy is Disabled",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *disableUserlandProxyBenchmark) Run() (result common.TestResult) {
	opts, ok := common.DockerConfig["userland-proxy"]
	if !ok {
		result.Warn()
		result.AddNotes("userland proxy is enabled by default")
		return
	}
	if opts.Matches("false") {
		result.Warn()
		result.AddNotes("userland proxy is enabled")
		return
	}
	result.Pass()
	return

}

// NewDisableUserlandProxyBenchmark implements CIS-2.15
func NewDisableUserlandProxyBenchmark() common.Benchmark {
	return &disableUserlandProxyBenchmark{}
}
