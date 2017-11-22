package dockerdaemonconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type baseDeviceSizeBenchmark struct{}

func (c *baseDeviceSizeBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 2.10",
			Description: "Ensure base device size is not changed until needed",
		}, Dependencies: []utils.Dependency{utils.InitDockerConfig},
	}
}

func (c *baseDeviceSizeBenchmark) Run() (result v1.BenchmarkTestResult) {
	opts, ok := utils.DockerConfig["storage-opt"]
	if ok {
		if val, found := opts.Contains("dm.basesize"); found {
			utils.Warn(&result)
			utils.AddNotes(&result, "Storage opt for basesize is %v", val)
			return
		}
	}
	utils.Pass(&result)
	return
}

// NewBaseDeviceSizeBenchmark implements CIS-2.10
func NewBaseDeviceSizeBenchmark() utils.Benchmark {
	return &baseDeviceSizeBenchmark{}
}
