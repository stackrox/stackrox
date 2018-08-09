package dockersecurityoperations

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type imageSprawlBenchmark struct{}

func (c *imageSprawlBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 6.1",
			Description: "Ensure image sprawl is avoided",
		}, Dependencies: []utils.Dependency{utils.InitImages, utils.InitContainers},
	}
}

func (c *imageSprawlBenchmark) Run() (result v1.CheckResult) {
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
