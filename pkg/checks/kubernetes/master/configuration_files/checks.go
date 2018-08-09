package masterconfigurationfiles

import (
	"github.com/stackrox/rox/pkg/checks"
	"github.com/stackrox/rox/pkg/checks/utils"
)

// NewServerPodSpecificationPermissions implements CIS Kubernetes v1.2.0 1.4.1
func NewServerPodSpecificationPermissions() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.1",
		"Ensure that the API server pod specification file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/manifests/kube-apiserver.yaml",
		0644,
		true,
	)
}

// NewServerPodSpecificationOwnership implements CIS Kubernetes v1.2.0 1.4.2
func NewServerPodSpecificationOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.2",
		"Ensure that the API server pod specification file ownership is set to root:root",
		"/etc/kubernetes/manifests/kube-apiserver.yaml",
		"root",
		"root",
	)
}

// NewControllerPodSpecificationOwnership implements CIS Kubernetes v1.2.0 1.4.4
func NewControllerPodSpecificationOwnership() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.3",
		"Ensure that the controller manager pod specification file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/manifests/kube-controller-manager.yaml",
		0644,
		true,
	)
}

// NewControllerPodSpecificationPermissions implements CIS Kubernetes v1.2.0 1.4.3
func NewControllerPodSpecificationPermissions() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.4",
		"Ensure that the controller manager pod specification file ownership is set to root:root",
		"/etc/kubernetes/manifests/kube-controller-manager.yaml",
		"root",
		"root",
	)
}

// NewSchedulerSpecificationPermissions implements CIS Kubernetes v1.2.0 1.4.5
func NewSchedulerSpecificationPermissions() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.5",
		"Ensure that the scheduler pod specification file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/manifests/kube-scheduler.yaml",
		0644,
		true,
	)
}

// NewSchedulerSpecificationOwnership implements CIS Kubernetes v1.2.0 1.4.6
func NewSchedulerSpecificationOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.6",
		"Ensure that the scheduler pod specification file ownership is set to root:root",
		"/etc/kubernetes/manifests/kube-scheduler.yaml",
		"root",
		"root",
	)
}

// NewEtcdSpecificationPermissions implements CIS Kubernetes v1.2.0 1.4.7
func NewEtcdSpecificationPermissions() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.7",
		"Ensure that the etcd pod specification file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/manifests/etcd.yaml",
		0644,
		true,
	)
}

// NewEtcdSpecificationOwnership implements CIS Kubernetes v1.2.0 1.4.8
func NewEtcdSpecificationOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.8",
		"Ensure that the etcd pod specification file ownership is set to root:root",
		"/etc/kubernetes/manifests/etcd.yaml",
		"root",
		"root",
	)
}

// 1.4.9 - 1.4.12 are unique

// NewAdminConfPermission implements CIS Kubernetes v1.2.0 1.4.13
func NewAdminConfPermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.13",
		"Ensure that the admin.conf file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/admin.conf",
		0644,
		true,
	)
}

// NewAdminConfOwnership implements CIS Kubernetes v1.2.0 1.4.14
func NewAdminConfOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.14",
		"Ensure that the admin.conf file ownership is set to root:root",
		"/etc/kubernetes/admin.conf",
		"root",
		"root",
	)
}

// NewSchedulerConfPermission implements CIS Kubernetes v1.2.0 1.4.15
func NewSchedulerConfPermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.15",
		"Ensure that the scheduler.conf file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/scheduler.conf",
		0644,
		true,
	)
}

// NewSchedulerConfOwnership implements CIS Kubernetes v1.2.0 1.4.16
func NewSchedulerConfOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.16",
		"Ensure that the scheduler.conf file ownership is set to root:root",
		"/etc/kubernetes/scheduler.conf",
		"root",
		"root",
	)
}

// NewControllerManagerConfPermission implements CIS Kubernetes v1.2.0 1.4.17
func NewControllerManagerConfPermission() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Kubernetes v1.2.0 - 1.4.17",
		"Ensure that the controller-manager.conf file permissions are set to 644 or more restrictive",
		"/etc/kubernetes/controller-manager.conf",
		0644,
		true,
	)
}

// NewControllerManagerConfOwnership implements CIS Kubernetes v1.2.0 1.4.18
func NewControllerManagerConfOwnership() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Kubernetes v1.2.0 - 1.4.18",
		"Ensure that the controller-manager.conf file ownership is set to root:root",
		"/etc/kubernetes/controller-manager.conf",
		"root",
		"root",
	)
}

func init() {
	checks.AddToRegistry(
		NewServerPodSpecificationPermissions(),
		NewServerPodSpecificationOwnership(),
		NewControllerPodSpecificationPermissions(),
		NewControllerPodSpecificationOwnership(),
		NewSchedulerSpecificationPermissions(),
		NewSchedulerSpecificationOwnership(),
		NewEtcdSpecificationPermissions(),
		NewEtcdSpecificationOwnership(),
		NewAdminConfPermission(),
		NewAdminConfOwnership(),
		NewSchedulerConfPermission(),
		NewSchedulerConfOwnership(),
		NewControllerManagerConfPermission(),
		NewControllerManagerConfOwnership(),
	)
}
