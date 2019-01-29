package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_1", "Ensure that the cluster-admin role is only used where required"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_2", "Create Pod Security Policies for your cluster"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_3", "Create administrative boundaries between resources using namespaces"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_4", "Create network segmentation using Network Policies"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_5", "Ensure that the seccomp profile is set to docker/default in your pod definitions"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_6", "Apply Security Context to Your Pods and Containers"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_7", "Configure Image Provenance using ImagePolicyWebhook admission controller"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_8", "Configure Network policies as appropriate"),
		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_6_9", "Place compensating controls in the form of PSP and RBAC for privileged containers usage"),
	)
}
