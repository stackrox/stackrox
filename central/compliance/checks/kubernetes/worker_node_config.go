package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PermissionCheck("CIS_Kubernetes_v1_5:4_1_1", "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", 0644),
		common.OwnershipCheck("CIS_Kubernetes_v1_5:4_1_2", "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", "root", "root"),

		common.CommandLineFilePermissions("CIS_Kubernetes_v1_5:4_1_3", "kubelet", "kubeconfig", 0644),
		common.CommandLineFileOwnership("CIS_Kubernetes_v1_5:4_1_4", "kubelet", "kubeconfig", "root", "root"),

		common.PermissionCheck("CIS_Kubernetes_v1_5:4_1_5", "/etc/kubernetes/kubelet.conf", 0644),
		common.OwnershipCheck("CIS_Kubernetes_v1_5:4_1_6", "/etc/kubernetes/kubelet.conf", "root", "root"),

		common.CommandLineFilePermissions("CIS_Kubernetes_v1_5:4_1_7", "kubelet", "client-ca-file", 0644),
		common.CommandLineFileOwnership("CIS_Kubernetes_v1_5:4_1_8", "kubelet", "client-ca-file", "root", "root"),

		common.CommandLineFilePermissions("CIS_Kubernetes_v1_5:4_1_9", "kubelet", "config", 0644),
		common.CommandLineFileOwnership("CIS_Kubernetes_v1_5:4_1_10", "kubelet", "config", "root", "root"),
	)
}
