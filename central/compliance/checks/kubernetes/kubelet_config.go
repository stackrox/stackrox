package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PermissionCheck("CIS_Kubernetes_v1_2_0:2_2_1", "/etc/kubernetes/kubelet.conf", 0644),
		common.OwnershipCheck("CIS_Kubernetes_v1_2_0:2_2_2", "/etc/kubernetes/kubelet.conf", "root", "root"),

		common.PermissionCheck("CIS_Kubernetes_v1_2_0:2_2_3", "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", 0644),
		common.OwnershipCheck("CIS_Kubernetes_v1_2_0:2_2_4", "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", "root", "root"),

		common.CommandLineFilePermissions("CIS_Kubernetes_v1_2_0:2_2_5", "kubelet", "kubeconfig", 0644),
		common.CommandLineFileOwnership("CIS_Kubernetes_v1_2_0:2_2_6", "kubelet", "kubeconfig", "root", "root"),

		common.CommandLineFilePermissions("CIS_Kubernetes_v1_2_0:2_2_7", "kubelet", "client-ca-file", 0644),
		common.CommandLineFileOwnership("CIS_Kubernetes_v1_2_0:2_2_8", "kubelet", "client-ca-file", "root", "root"),
	)
}
