package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("4_1_1"): common.OptionalPermissionCheck("/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", 0644),
		standards.CISKubeCheckName("4_1_2"): common.OptionalOwnershipCheck("/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", "root", "root"),

		standards.CISKubeCheckName("4_1_3"): common.CommandLineFilePermissions("kubelet", "kubeconfig", 0644),
		standards.CISKubeCheckName("4_1_4"): common.CommandLineFileOwnership("kubelet", "kubeconfig", "root", "root"),

		standards.CISKubeCheckName("4_1_5"): common.OptionalPermissionCheck("/etc/kubernetes/kubelet.conf", 0644),
		standards.CISKubeCheckName("4_1_6"): common.OptionalOwnershipCheck("/etc/kubernetes/kubelet.conf", "root", "root"),

		standards.CISKubeCheckName("4_1_7"): common.CommandLineFilePermissions("kubelet", "client-ca-file", 0644),
		standards.CISKubeCheckName("4_1_8"): common.CommandLineFileOwnership("kubelet", "client-ca-file", "root", "root"),

		standards.CISKubeCheckName("4_1_9"):  common.CommandLineFilePermissions("kubelet", "config", 0644),
		standards.CISKubeCheckName("4_1_10"): common.CommandLineFileOwnership("kubelet", "config", "root", "root"),
	})
}
