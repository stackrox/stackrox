package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type imageHealthcheckBenchmark struct{}

func (c *imageHealthcheckBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.6",
		Description:  "Ensure HEALTHCHECK instructions have been added to the container image",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *imageHealthcheckBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, image := range common.Images {
		if image.Config.Healthcheck == nil {
			result.Warn()
			result.AddNotef("Image %v does not have healthcheck configured", common.GetReadableImageName(image))
		}
	}
	return
}

// NewImageHealthcheckBenchmark implements CIS-4.6
func NewImageHealthcheckBenchmark() common.Benchmark {
	return &imageHealthcheckBenchmark{}
}
