package containerimagesandbuild

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type imageHealthcheckBenchmark struct{}

func (c *imageHealthcheckBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        "CIS 4.6",
			Description: "Ensure HEALTHCHECK instructions have been added to the container image",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageHealthcheckBenchmark) Run() (result v1.BenchmarkTestResult) {
	utils.Pass(&result)
	for _, image := range utils.Images {
		if image.Config.Healthcheck == nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Image %v does not have healthcheck configured", utils.GetReadableImageName(image))
		}
	}
	return
}

// NewImageHealthcheckBenchmark implements CIS-4.6
func NewImageHealthcheckBenchmark() utils.Benchmark {
	return &imageHealthcheckBenchmark{}
}
