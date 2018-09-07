package securityprimitives

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

// TODO(cgorman) is there anything more we can for these checks other than print the description

// NewClusterAdmin implements CIS Kubernetes v1.2.0 1.6.1
func NewClusterAdmin() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.1",
		Description: "Ensure that the cluster-admin role is only used where required",

		EvalFunc: utils.Skip,
	}
}

// NewPodSecurityPolicies implements CIS Kubernetes v1.2.0 1.6.2
func NewPodSecurityPolicies() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.2",
		Description: "Create Pod Security Policies for your cluster",

		EvalFunc: utils.Skip,
	}
}

// NewAdminBoundaries implements CIS Kubernetes v1.2.0 1.6.3
func NewAdminBoundaries() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.3",
		Description: "Create administrative boundaries between resources using namespaces",

		EvalFunc: utils.Skip,
	}
}

// NewNetworkSegmentation implements CIS Kubernetes v1.2.0 1.6.4
func NewNetworkSegmentation() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.4",
		Description: "Create network segmentation using Network Policies",

		EvalFunc: utils.Skip,
	}
}

// NewSeccompProfile implements CIS Kubernetes v1.2.0 1.6.5
func NewSeccompProfile() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.5",
		Description: "Ensure that the seccomp profile is set to docker/default in your pod definitions",

		EvalFunc: utils.Skip,
	}
}

// NewSecurityContext implements CIS Kubernetes v1.2.0 1.6.6
func NewSecurityContext() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.6",
		Description: "Apply Security Context to Your Pods and Containers",

		EvalFunc: utils.Skip,
	}
}

// NewImagePolicyWebhook implements CIS Kubernetes v1.2.0 1.6.7
func NewImagePolicyWebhook() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.7",
		Description: "Configure Image Provenance using ImagePolicyWebhook admission controller",

		EvalFunc: utils.Skip,
	}
}

// NewNetworkPolicies implements CIS Kubernetes v1.2.0 1.6.8
func NewNetworkPolicies() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.8",
		Description: "Configure Network policies as appropriate",

		EvalFunc: utils.Skip,
	}
}

// NewPrivileged implements CIS Kubernetes v1.2.0 1.6.9
func NewPrivileged() utils.Check {
	return &utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.6.9",
		Description: "Place compensating controls in the form of PSP and RBAC for privileged containers usage",

		EvalFunc: utils.Skip,
	}
}

func init() {
	checks.AddToRegistry(
		NewClusterAdmin(),
		NewPodSecurityPolicies(),
		NewAdminBoundaries(),
		NewNetworkSegmentation(),
		NewSeccompProfile(),
		NewSecurityContext(),
		NewImagePolicyWebhook(),
		NewNetworkPolicies(),
		NewPrivileged(),
	)
}
