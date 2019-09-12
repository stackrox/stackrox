package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_1", "Do not admit privileged containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_2", "Do not admit containers wishing to share the host process ID namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_3", "Do not admit containers wishing to share the host IPC namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_4", "Do not admit containers wishing to share the host network namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_5", "Do not admit containers with allowPrivilegeEscalation"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_6", "Do not admit root containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_4_1:1_7_7", "Do not admit containers with dangerous capabilities"),
	)
}
