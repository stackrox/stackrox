package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:3_1_1", "Client certificate authentication should not be used for users"),
		masterAPIServerCommandLine("CIS_Kubernetes_v1_5:3_2_1", "--audit-policy-file", "", "", common.Set),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:3_2_2", "Ensure that the audit policy covers key security concerns"),
	)
}
