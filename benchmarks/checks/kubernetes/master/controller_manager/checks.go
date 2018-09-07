package controllermanager

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

const process = "kube-controller-manager"

var configFunc = utils.GetKubeControllerManagerConfig

func newKubernetesControllerManagerCheck(check *utils.CommandCheck) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

func newMultipleKubernetesControllerManagerCheck(check *utils.MultipleCommandChecks) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

// NewTerminatedPodGCThreshold implements CIS Kubernetes v1.2.0 1.3.1
func NewTerminatedPodGCThreshold() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{

		Name:        "CIS Kubernetes v1.2.0 - 1.3.1",
		Description: "Ensure that the --terminated-pod-gc-threshold argument is set as appropriate",

		Field:    "terminated-pod-gc-threshold",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewProfiling implements CIS Kubernetes v1.2.0 1.3.2
func NewProfiling() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.2",
		Description: "Ensure that the --profiling argument is set to false",

		Field:        "profiling",
		Default:      "true",
		EvalFunc:     utils.Contains,
		DesiredValue: "false",
	})
}

// NewServiceAccountCreds implements CIS Kubernetes v1.2.0 1.3.3
func NewServiceAccountCreds() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.3",
		Description: "Ensure that the --use-service-account-credentials argument is set to true",

		Field:        "use-service-account-credentials",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewServiceAccountPrivateKey implements CIS Kubernetes v1.2.0 1.3.4
func NewServiceAccountPrivateKey() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.4",
		Description: "Ensure that the --service-account-private-key-file argument is set as appropriate",

		Field:    "service-account-private-key-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewRootCAFile implements CIS Kubernetes v1.2.0 1.3.5
func NewRootCAFile() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.5",
		Description: "Ensure that the --root-ca-file argument is set as appropriate",

		Field:    "root-ca-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewSecurityContext implements CIS Kubernetes v1.2.0 1.3.5
func NewSecurityContext() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.6",
		Description: "Apply Security Context to Your Pods and Containers",

		EvalFunc: utils.Skip,
	})
}

// NewRotateKubeletServiceCert implements CIS Kubernetes v1.2.0 1.3.6
func NewRotateKubeletServiceCert() utils.Check {
	return newKubernetesControllerManagerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.3.7",
		Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",

		Field:        "feature-gates",
		EvalFunc:     utils.Contains,
		DesiredValue: "RotateKubeletServerCertificate=true",
	})
}

func init() {
	checks.AddToRegistry(
		NewTerminatedPodGCThreshold(),
		NewProfiling(),
		NewServiceAccountCreds(),
		NewServiceAccountPrivateKey(),
		NewRootCAFile(),
		NewSecurityContext(),
		NewRotateKubeletServiceCert(),
	)
}
