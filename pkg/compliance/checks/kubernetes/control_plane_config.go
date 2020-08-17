package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("3_1_1"): common.NoteCheck("Client certificate authentication should not be used for users"),
		standards.CISKubeCheckName("3_2_1"): common.MasterAPIServerCommandLine("--audit-policy-file", "", "", common.Set),
		standards.CISKubeCheckName("3_2_2"): common.NoteCheck("Ensure that the audit policy covers key security concerns"),
	})
}
