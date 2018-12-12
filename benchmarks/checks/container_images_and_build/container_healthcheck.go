package containerimagesandbuild

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type imageHealthcheckBenchmark struct{}

func (c *imageHealthcheckBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.6",
			Description: "Ensure HEALTHCHECK instructions have been added to the container image",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageHealthcheckBenchmark) Run() (result storage.BenchmarkCheckResult) {
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
func NewImageHealthcheckBenchmark() utils.Check {
	return &imageHealthcheckBenchmark{}
}
