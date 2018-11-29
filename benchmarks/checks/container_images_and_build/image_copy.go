package containerimagesandbuild

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
)

type imageCopyBenchmark struct{}

func (c *imageCopyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 4.9",
			Description: "Ensure COPY is used instead of ADD in Dockerfile",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageCopyBenchmark) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, image := range utils.Images {
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		historySlice, err := utils.DockerClient.ImageHistory(ctx, image.ID)
		if err != nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Could not get image history for image %v: %s", image.ID, err)
			continue
		}
		for _, history := range historySlice {
			cmd := strings.ToLower(history.CreatedBy)
			if strings.Contains(cmd, "add file:") || strings.Contains(cmd, "add dir:") {
				utils.Warn(&result)
				utils.AddNotef(&result, "Image %v has an ADD instead of a COPY command", utils.GetReadableImageName(image))
				break
			}
		}
	}
	return
}

// NewImageCopyBenchmark implements CIS-4.9
func NewImageCopyBenchmark() utils.Check {
	return &imageCopyBenchmark{}
}
