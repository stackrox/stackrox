package containerimagesandbuild

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type imageCopyBenchmark struct{}

func (c *imageCopyBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.9",
		Description:  "Ensure COPY is used instead of ADD in Dockerfile",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *imageCopyBenchmark) Run() (result common.TestResult) {
	result.Pass()
	for _, image := range common.Images {
		historySlice, err := common.DockerClient.ImageHistory(context.Background(), image.ID)
		if err != nil {
			result.Warn()
			result.AddNotef("Could not get image history for image %v: %+v", err)
			continue
		}
		for _, history := range historySlice {
			cmd := strings.ToLower(history.CreatedBy)
			if strings.Contains(cmd, "add file:") || strings.Contains(cmd, "add dir:") {
				result.Warn()
				result.AddNotef("Image %v has an ADD instead of a COPY command", common.GetReadableImageName(image))
				break
			}
		}
	}
	return
}

// NewImageCopyBenchmark implements CIS-4.9
func NewImageCopyBenchmark() common.Benchmark {
	return &imageCopyBenchmark{}
}
