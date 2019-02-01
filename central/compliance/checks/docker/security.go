package docker

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/set"
)

func init() {
	framework.MustRegisterChecks(
		framework.NewCheckFromFunc("CIS_Docker_v1_1_0:6_1", framework.NodeKind, nil, imageSprawl),
		framework.NewCheckFromFunc("CIS_Docker_v1_1_0:6_2", framework.NodeKind, nil, containerSprawl),
	)
}

func imageSprawl(funcCtx framework.ComplianceContext) {
	perNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *docker.Data) {
		imageSet := set.NewStringSet()
		for _, c := range data.Containers {
			imageSet.Add(c.Image)
		}
		framework.Notef(ctx, "There are %d images in use out of %d", imageSet.Cardinality(), len(data.Images))
	})(funcCtx)
}

func containerSprawl(funcCtx framework.ComplianceContext) {
	perNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *docker.Data) {
		var runningContainers int
		for _, c := range data.Containers {
			if c.State != nil && c.State.Running == true {
				runningContainers++
			}
		}
		framework.Notef(ctx, "There are %d containers in use out of %d", runningContainers, len(data.Containers))
	})(funcCtx)
}
