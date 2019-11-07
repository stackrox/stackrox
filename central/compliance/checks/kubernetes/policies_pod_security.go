package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_1", "Minimize the admission of privileged containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_2", "Minimize the admission of containers wishing to share the host process ID namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_3", "Minimize the admission of containers wishing to share the host IPC namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_4", "Minimize the admission of containers wishing to share the host network namespace"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_5", "Minimize the admission of containers with allowPrivilegeEscalation"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_6", "Minimize the admission of root containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_7", "Minimize the admission of containers with the NET_RAW capability"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_8", "Minimize the admission of containers with added capabilities"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_2_9", "Minimize the admission of containers with capabilities assigned"),
	)
}
