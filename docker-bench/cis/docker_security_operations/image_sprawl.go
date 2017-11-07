package dockersecurityoperations

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type imageSprawlBenchmark struct{}

func (c *imageSprawlBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 6.1",
		Description:  "Ensure image sprawl is avoided",
		Dependencies: []common.Dependency{common.InitImages, common.InitContainers},
	}
}

func (c *imageSprawlBenchmark) Run() (result common.TestResult) {
	result.Info()
	m := make(map[string]struct{})
	for _, container := range common.ContainersRunning {
		m[container.Image] = struct{}{}
	}
	result.AddNotef("There are %v images in use out of %v", len(m), len(common.Images))
	return
}

// NewImageSprawlBenchmark implements CIS-6.1
func NewImageSprawlBenchmark() common.Benchmark {
	return &imageSprawlBenchmark{}
}
