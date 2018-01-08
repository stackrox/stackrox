package containerimagesandbuild

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
)

type imageCopyBenchmark struct{}

func (c *imageCopyBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.9",
			Description: "Ensure COPY is used instead of ADD in Dockerfile",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageCopyBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, image := range utils.Images {
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		historySlice, err := utils.DockerClient.ImageHistory(ctx, image.ID)
		if err != nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Could not get image history for image %v: %+v", err)
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
