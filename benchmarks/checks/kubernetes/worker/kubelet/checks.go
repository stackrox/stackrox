package kubelet

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

const process = "kubelet"

var configFunc = utils.GetKubeletConfig

func newKubeletCheck(check *utils.CommandCheck) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

func newMultipleKubeletCheck(check *utils.MultipleCommandChecks) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

// NewAllowPrivileged implements CIS Kubernetes v1.2.0 2.1.1
func NewAllowPrivileged() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.1",
		Description: "Ensure that the --allow-privileged argument is set to false",

		Field:        "allow-privileged",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewAnonymousAuth implements CIS Kubernetes v1.2.0 2.1.2
func NewAnonymousAuth() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.2",
		Description: "Ensure that the --anonymous-auth argument is set to false",

		Field:        "anonymous-auth",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewAuthorizationMode implements CIS Kubernetes v1.2.0 2.1.3
func NewAuthorizationMode() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.3",
		Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",

		Field:        "authorization-mode",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AlwaysAllow",
	})
}

// NewClientCAFile implements CIS Kubernetes v1.2.0 2.1.4
func NewClientCAFile() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.4",
		Description: "Ensure that the --client-ca-file argument is set as appropriate",

		Field:    "client-ca-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewReadOnlyPort implements CIS Kubernetes v1.2.0 2.1.5
func NewReadOnlyPort() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.5",
		Description: "Ensure that the --read-only-port argument is set to 0",

		Field:        "read-only-port",
		Default:      "0",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// NewStreamingConnectionIdleTimeout implements CIS Kubernetes v1.2.0 2.1.6
func NewStreamingConnectionIdleTimeout() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.6",
		Description: "Ensure that the --streaming-connection-idle-timeout argument is not set to 0",

		Field:        "streaming-connection-idle-timeout",
		Default:      "0",
		EvalFunc:     utils.NotMatches,
		DesiredValue: "0",
	})
}

// NewProtectKernelDefaults implements CIS Kubernetes v1.2.0 2.1.7
func NewProtectKernelDefaults() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.7",
		Description: "Ensure that the --protect-kernel-defaults argument is set to true",

		Field:        "protect-kernel-defaults",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewMakeIptablesUtilChains implements CIS Kubernetes v1.2.0 2.1.8
func NewMakeIptablesUtilChains() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.8",
		Description: "Ensure that the --make-iptables-util-chains argument is set to true",

		Field:        "make-iptables-util-chains",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewKeepTerminatedPod implements CIS Kubernetes v1.2.0 2.1.9
func NewKeepTerminatedPod() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.9",
		Description: "Ensure that the --keep-terminated-pod-volumes argument is set to false",

		Field:        "keep-terminated-pod-volumes",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewHostnameOverride implements CIS Kubernetes v1.2.0 2.1.10
func NewHostnameOverride() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.10",
		Description: "Ensure that the --hostname-override argument is not set",

		Field:    "hostname-override",
		EvalFunc: utils.Unset,
	})
}

// NewEventQPS implements CIS Kubernetes v1.2.0 2.1.11
func NewEventQPS() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.11",
		Description: "Ensure that the --event-qps argument is set to 0",

		Field:        "event-qps",
		Default:      "5",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// NewTLSCertFiles implements CIS Kubernetes v1.2.0 2.1.12
func NewTLSCertFiles() utils.Check {
	return newMultipleKubeletCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.12",
		Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "tls-cert-file",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "tls-private-key-file",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewCadvisorPort implements CIS Kubernetes v1.2.0 2.1.13
func NewCadvisorPort() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.13",
		Description: "Ensure that the --cadvisor-port argument is set to 0",

		Field:        "cadvisor-port",
		Default:      "0",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// NewRotateKubeletClientCert implements CIS Kubernetes v1.2.0 2.1.14
func NewRotateKubeletClientCert() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.14",
		Description: "Ensure that the RotateKubeletClientCertificate argument is not set to false",

		Field:        "feature-gates",
		Default:      "RotateKubeletClientCertificate=true",
		EvalFunc:     utils.NotContains,
		DesiredValue: "RotateKubeletClientCertificate=false",
	})
}

// NewRotateKubeletServerCert implements CIS Kubernetes v1.2.0 2.1.15
func NewRotateKubeletServerCert() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 2.1.15",
		Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",

		Field:        "feature-gates",
		Default:      "RotateKubeletServerCertificate=false",
		EvalFunc:     utils.Contains,
		DesiredValue: "RotateKubeletServerCertificate=true",
	})
}

func init() {
	checks.AddToRegistry(
		NewAllowPrivileged(),
		NewAnonymousAuth(),
		NewAuthorizationMode(),
		NewClientCAFile(),
		NewReadOnlyPort(),
		NewStreamingConnectionIdleTimeout(),
		NewProtectKernelDefaults(),
		NewMakeIptablesUtilChains(),
		NewKeepTerminatedPod(),
		NewHostnameOverride(),
		NewEventQPS(),
		NewTLSCertFiles(),
		NewCadvisorPort(),
		NewRotateKubeletClientCert(),
		NewRotateKubeletServerCert(),
	)
}
