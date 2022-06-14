package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_1_1"): common.NoteCheck("Ensure that the cluster-admin role is only used where required"),
		standards.CISKubeCheckName("5_1_2"): common.NoteCheck("Minimize access to secrets"),
		standards.CISKubeCheckName("5_1_3"): common.NoteCheck("Minimize wildcard use in Roles and ClusterRoles"),
		standards.CISKubeCheckName("5_1_4"): common.NoteCheck("Minimize access to create pods"),
		standards.CISKubeCheckName("5_1_5"): common.NoteCheck("Ensure that default service accounts are not actively used"),
		standards.CISKubeCheckName("5_1_6"): common.NoteCheck("Ensure that Service Account Tokens are only mounted where necessary"),
	})
}
