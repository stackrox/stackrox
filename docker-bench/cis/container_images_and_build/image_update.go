package containerimagesandbuild

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type imageUpdateInstructionsBenchmark struct{}

func (c *imageUpdateInstructionsBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 4.7",
			Description: "Ensure update instructions are not use alone in the Dockerfile",
		}, Dependencies: []utils.Dependency{utils.InitImages},
	}
}

func (c *imageUpdateInstructionsBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	for _, image := range utils.Images {
		historySlice, err := utils.DockerClient.ImageHistory(context.Background(), image.ID)
		if err != nil {
			utils.Warn(&result)
			utils.AddNotef(&result, "Could not get image history for image %v: %+v", err)
			continue
		}
		for _, history := range historySlice {
			cmd := strings.ToLower(history.CreatedBy)
			if strings.Contains(cmd, "update") && !strings.Contains(cmd, "&&") {
				utils.Warn(&result)
				utils.AddNotef(&result, "Image %v has an update command alone in layer: %v", utils.GetReadableImageName(image), history.ID, cmd)
			}
		}
	}
	return
}

// NewImageUpdateInstructionsBenchmark implements CIS-4.7
func NewImageUpdateInstructionsBenchmark() utils.Check {
	return &imageUpdateInstructionsBenchmark{}
}
