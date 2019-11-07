package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_1", "Ensure that the cluster-admin role is only used where required"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_2", "Minimize access to secrets"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_3", "Minimize wildcard use in Roles and ClusterRoles"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_4", "Minimize access to create pods"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_5", "Ensure that default service accounts are not actively used"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_1_6", "Ensure that Service Account Tokens are only mounted where necessary"),
	)
}
