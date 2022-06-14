package docker

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/compliance/msgfmt"
	"github.com/stackrox/rox/pkg/docker/types"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		// 4_1 is in runtime.go
		standards.CISDockerCheckName("4_2"):  common.NoteCheck("Verify that only trusted base images are used"),
		standards.CISDockerCheckName("4_3"):  common.NoteCheck("Check if the packages inside the image are necessary"),
		standards.CISDockerCheckName("4_4"):  common.NoteCheck("Check if images are scanned and rebuilt to include security patches"),
		standards.CISDockerCheckName("4_5"):  common.NoteCheck("Docker content trust is set on an individual basis via environment variable"),
		standards.CISDockerCheckName("4_6"):  imageCheck(healthcheckInstruction, "has a health check configured"),
		standards.CISDockerCheckName("4_7"):  imageCheck(noUpdateInstruction, "does not use update commands such as `apt-get update`"),
		standards.CISDockerCheckName("4_8"):  common.NoteCheck("Check if setuid and setgid permissions are removed in the images"),
		standards.CISDockerCheckName("4_9"):  imageCheck(copyInstruction, "uses COPY instead of ADD"),
		standards.CISDockerCheckName("4_10"): common.NoteCheck("Ensure secrets are not stored in Dockerfiles"),
		standards.CISDockerCheckName("4_11"): common.NoteCheck("Check if only verified packages are installed"),
	})
}

func imageCheck(f func(wrap types.ImageWrap) []*storage.ComplianceResultValue_Evidence, desc string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: common.CheckWithDockerData(func(data *types.Data) []*storage.ComplianceResultValue_Evidence {
			var checkResults []*storage.ComplianceResultValue_Evidence
			for _, i := range data.Images {
				checkResults = append(checkResults, f(i)...)
			}
			return checkResults
		}),
		Metadata: &standards.Metadata{
			InterpretationText: fmt.Sprintf("StackRox checks that each image on every node %s", desc),
			TargetKind:         framework.NodeKind,
		},
	}
}

func healthcheckInstruction(wrap types.ImageWrap) []*storage.ComplianceResultValue_Evidence {
	if wrap.Config().Healthcheck == nil {
		return common.FailListf("Image %q does not have healthcheck configured", wrap.Name())
	}
	return common.PassListf("Image %q implements healthcheck with test: %s", wrap.Name(), msgfmt.FormatStrings(wrap.Config().Healthcheck.Test...))
}

func copyInstruction(wrap types.ImageWrap) []*storage.ComplianceResultValue_Evidence {
	var fail bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, h := range wrap.History {
		cmd := strings.ToLower(h.CreatedBy)
		if strings.Contains(cmd, "add file:") || strings.Contains(cmd, "add dir:") {
			fail = true
			results = append(results, common.Failf("Image %q has a Dockerfile line %q that uses an ADD instead of a COPY", wrap.Name(), cmd))
		}
	}
	if !fail {
		results = append(results, common.Passf("Image %q does not have a Dockerfile that uses ADD", wrap.Name()))
	}
	return results
}

var updateCmds = []string{
	"apk update",
	"apt update",
	"apt-get update",
	"yum update",
}

func noUpdateInstruction(wrap types.ImageWrap) []*storage.ComplianceResultValue_Evidence {
	var fail bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, h := range wrap.History {
		cmd := strings.ToLower(h.CreatedBy)
		cmd = strings.Replace(cmd, "\t", "", -1)
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c #(nop)")
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c #")
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c")
		cmd = strings.TrimSpace(cmd)
		for _, updateCmd := range updateCmds {
			if cmd == updateCmd {
				fail = true
				results = append(results, common.Failf("Image %q has an update command %q", wrap.Name(), cmd))
			}
		}
	}
	if !fail {
		results = append(results, common.Passf("Image %q does not have an update command", wrap.Name()))
	}
	return results
}
