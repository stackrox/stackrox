package docker

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/docker"
)

func init() {
	framework.MustRegisterChecks(
		// 4_1 is in runtime.go
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_2", "Verify that only trusted base images are used"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_3", "Check if the packages inside the image are necessary"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_4", "Check if images are scanned"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_5", "Docker content trust is set on an individual basis via environment variable"),
		imageCheck("CIS_Docker_v1_1_0:4_6", healthcheckInstruction, "has a health check configured"),
		imageCheck("CIS_Docker_v1_1_0:4_7", noUpdateInstruction, "does not use update commands such as `apt-get update`"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_8", "Check if setuid and setgid permissions are removed in the images"),
		imageCheck("CIS_Docker_v1_1_0:4_9", copyInstruction, "uses COPY instead of ADD"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_10", "Ensure secrets are not stored in Dockerfiles"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:4_11", "Check if only verified packages are installed"),
	)
}

func imageCheck(name string, f func(ctx framework.ComplianceContext, wrap docker.ImageWrap), desc string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that each image on every node %s", desc),
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, imageCheckWrapper(f))
}

func imageCheckWrapper(f func(ctx framework.ComplianceContext, wrap docker.ImageWrap)) framework.CheckFunc {
	return common.PerNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *docker.Data) {
		for _, i := range data.Images {
			f(ctx, i)
		}
	})
}

func healthcheckInstruction(ctx framework.ComplianceContext, wrap docker.ImageWrap) {
	if wrap.Config().Healthcheck == nil {
		framework.Failf(ctx, "Image %q does not have healthcheck configured", wrap.Name())
	} else {
		framework.Passf(ctx, "Image %q implements healthcheck with test: %s", wrap.Name(), msgfmt.FormatStrings(wrap.Config().Healthcheck.Test...))
	}
}

func copyInstruction(ctx framework.ComplianceContext, wrap docker.ImageWrap) {
	var fail bool
	for _, h := range wrap.History {
		cmd := strings.ToLower(h.CreatedBy)
		if strings.Contains(cmd, "add file:") || strings.Contains(cmd, "add dir:") {
			fail = true
			framework.Failf(ctx, "Image %q has a Dockerfile line %q that uses an ADD instead of a COPY", wrap.Name(), cmd)
		}
	}
	if !fail {
		framework.Passf(ctx, "Image %q does not have a Dockerfile that uses ADD", wrap.Name())
	}
}

var updateCmds = []string{
	"apk update",
	"apt update",
	"apt-get update",
	"yum update",
}

func noUpdateInstruction(ctx framework.ComplianceContext, wrap docker.ImageWrap) {
	var fail bool
	for _, h := range wrap.History {
		cmd := strings.ToLower(h.CreatedBy)
		cmd = strings.Replace(cmd, "\t", "", -1)
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c #(nop)")
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c")
		cmd = strings.TrimSpace(cmd)
		for _, updateCmd := range updateCmds {
			if cmd == updateCmd {
				fail = true
				framework.Failf(ctx, "Image %q has an update command %q", wrap.Name(), cmd)
			}
		}
	}
	if !fail {
		framework.Passf(ctx, "Image %q does not have an update command", wrap.Name())
	}
}
