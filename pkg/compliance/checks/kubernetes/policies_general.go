package kubernetes

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_6_1"): common.NoteCheck("Create administrative boundaries between resources using namespaces"),
		standards.CISKubeCheckName("5_6_2"): common.NoteCheck("Ensure that the seccomp profile is set to docker/default in your pod definitions"),
		standards.CISKubeCheckName("5_6_3"): common.NoteCheck("Apply Security Context to Your Pods and Containers"),
		standards.CISKubeCheckName("5_6_4"): common.NoteCheck("The default namespace should not be used"),
	})
}
