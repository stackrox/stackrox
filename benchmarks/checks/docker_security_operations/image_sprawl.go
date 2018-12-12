package dockersecurityoperations

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type imageSprawlBenchmark struct{}

func (c *imageSprawlBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 6.1",
			Description: "Ensure image sprawl is avoided",
		}, Dependencies: []utils.Dependency{utils.InitImages, utils.InitContainers},
	}
}

func (c *imageSprawlBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Info(&result)
	m := make(map[string]struct{})
	for _, container := range utils.ContainersRunning {
		m[container.Image] = struct{}{}
	}
	utils.AddNotef(&result, "There are '%v' images in use out of '%v'", len(m), len(utils.Images))
	return
}

// NewImageSprawlBenchmark implements CIS-6.1
func NewImageSprawlBenchmark() utils.Check {
	return &imageSprawlBenchmark{}
}
