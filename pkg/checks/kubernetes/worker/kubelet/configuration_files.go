package kubelet

import (
	"github.com/stackrox/rox/pkg/checks"
	"github.com/stackrox/rox/pkg/checks/utils"
)

// NewKubeletConfPermission implements CIS Kubernetes v1.2.0 2.2.1
func NewKubeletConfPermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 2.2.1",
		"Ensure that the kubelet.conf file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/kubelet.conf",
		0644,
		true,
	)
}

// NewKubeletConfOwnership implements CIS Kubernetes v1.2.0 2.2.2
func NewKubeletConfOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 2.2.2",
		"Ensure that the kubelet.conf file ownership is set to root:root",
		"/etc/kubernetes/kubelet.conf",
		"root",
		"root",
	)
}

// NewKubeletServicePermission implements CIS Kubernetes v1.2.0 2.2.3
func NewKubeletServicePermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 2.2.3",
		"Ensure that the kubelet service file permissions are set to 644 or more restrictive",
		"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
		0644,
		true,
	)
}

// NewKubeletServiceOwnership implements CIS Kubernetes v1.2.0 2.2.4
func NewKubeletServiceOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 2.2.4",
		"Ensure that the kubelet service file ownership is set to root:root",
		"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
		"root",
		"root",
	)
}

// 2.2.5 - 2.2.8 are unique

func init() {
	checks.AddToRegistry(
		NewKubeletConfPermission(),
		NewKubeletConfOwnership(),
		NewKubeletServicePermission(),
		NewKubeletServiceOwnership(),
	)
}
