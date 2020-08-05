package docker

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/set"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		standards.CISDockerCheckName("6_1"): {
			CheckFunc: common.CheckWithDockerData(imageSprawl),
			Metadata: &standards.Metadata{
				InterpretationText: "StackRox checks how many of the images present on each node are actually in use",
				TargetKind:         framework.NodeKind,
			},
		},
		standards.CISDockerCheckName("6_2"): {
			CheckFunc: common.CheckWithDockerData(containerSprawl),
			Metadata: &standards.Metadata{
				InterpretationText: "StackRox checks how many of the containers present on each node are actually running",
				TargetKind:         framework.NodeKind,
			},
		},
	})
}

func imageSprawl(data *internalTypes.Data) []*storage.ComplianceResultValue_Evidence {
	imageSet := set.NewStringSet()
	for _, c := range data.Containers {
		imageSet.Add(c.Image)
	}
	return common.NoteListf("There are %d images in use out of %d", imageSet.Cardinality(), len(data.Images))
}

func containerSprawl(data *internalTypes.Data) []*storage.ComplianceResultValue_Evidence {
	var runningContainers int
	for _, c := range data.Containers {
		if c.State != nil && c.State.Running {
			runningContainers++
		}
	}
	return common.NoteListf("There are %d containers in use out of %d", runningContainers, len(data.Containers))
}
