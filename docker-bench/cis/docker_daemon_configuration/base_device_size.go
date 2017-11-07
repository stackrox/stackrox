package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type baseDeviceSizeBenchmark struct{}

func (c *baseDeviceSizeBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.10",
		Description:  "Ensure base device size is not changed until needed",
		Dependencies: []common.Dependency{common.InitDockerConfig},
	}
}

func (c *baseDeviceSizeBenchmark) Run() (result common.TestResult) {
	opts, ok := common.DockerConfig["storage-opt"]
	if ok {
		if val, found := opts.Contains("dm.basesize"); found {
			result.Result = common.Warn
			result.AddNotes("Storage opt for basesize is %v", val)
			return
		}
	}
	result.Result = common.Pass
	return
}

// NewBaseDeviceSizeBenchmark implements CIS-2.10
func NewBaseDeviceSizeBenchmark() common.Benchmark {
	return &baseDeviceSizeBenchmark{}
}
