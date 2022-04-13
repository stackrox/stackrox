package kubernetes

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_2_1"): common.NoteCheck("Minimize the admission of privileged containers"),
		standards.CISKubeCheckName("5_2_2"): common.NoteCheck("Minimize the admission of containers wishing to share the host process ID namespace"),
		standards.CISKubeCheckName("5_2_3"): common.NoteCheck("Minimize the admission of containers wishing to share the host IPC namespace"),
		standards.CISKubeCheckName("5_2_4"): common.NoteCheck("Minimize the admission of containers wishing to share the host network namespace"),
		standards.CISKubeCheckName("5_2_5"): common.NoteCheck("Minimize the admission of containers with allowPrivilegeEscalation"),
		standards.CISKubeCheckName("5_2_6"): common.NoteCheck("Minimize the admission of root containers"),
		standards.CISKubeCheckName("5_2_7"): common.NoteCheck("Minimize the admission of containers with the NET_RAW capability"),
		standards.CISKubeCheckName("5_2_8"): common.NoteCheck("Minimize the admission of containers with added capabilities"),
		standards.CISKubeCheckName("5_2_9"): common.NoteCheck("Minimize the admission of containers with capabilities assigned"),
	})
}
