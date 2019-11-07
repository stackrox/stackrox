package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_6_1", "Create administrative boundaries between resources using namespaces"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_6_2", "Ensure that the seccomp profile is set to docker/default in your pod definitions"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_6_3", "Apply Security Context to Your Pods and Containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_5:5_6_4", "The default namespace should not be used"),
	)
}
