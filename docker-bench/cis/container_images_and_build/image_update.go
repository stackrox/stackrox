package containerimagesandbuild

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type imageUpdateInstructionsBenchmark struct{}

func (c *imageUpdateInstructionsBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 4.7",
		Description:  "Ensure update instructions are not use alone in the Dockerfile",
		Dependencies: []common.Dependency{common.InitImages},
	}
}

func (c *imageUpdateInstructionsBenchmark) Run() (result common.TestResult) {
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
			if strings.Contains(cmd, "update") && !strings.Contains(cmd, "&&") {
				result.Warn()
				result.AddNotef("Image %v has an update command alone in layer: %v", common.GetReadableImageName(image), history.ID, cmd)
			}
		}
	}
	return
}

// NewImageUpdateInstructionsBenchmark implements CIS-4.7
func NewImageUpdateInstructionsBenchmark() common.Benchmark {
	return &imageUpdateInstructionsBenchmark{}
}
