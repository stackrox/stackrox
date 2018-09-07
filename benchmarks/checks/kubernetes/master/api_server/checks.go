package apiserver

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

func newKubernetesAPIServerCheck(check *utils.CommandCheck) utils.Check {
	check.Process = "kube-apiserver"
	check.ConfigGetter = utils.GetKubeAPIServerConfig
	return check
}

func newMultipleKubernetesAPIServerCheck(check *utils.MultipleCommandChecks) utils.Check {
	check.Process = "kube-apiserver"
	check.ConfigGetter = utils.GetKubeAPIServerConfig
	return check
}

// NewAnonymousAuth implements CIS Kubernetes v1.2.0 1.1.1
func NewAnonymousAuth() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.1",
		Description: "Ensure that the --anonymous-auth argument is set to false",

		Field:        "anonymous-auth",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewBasicAuthFile implements CIS Kubernetes v1.2.0 1.1.2
func NewBasicAuthFile() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.2",
		Description: "Ensure that the --basic-auth-file argument is not set",

		Field:    "basic-auth-file",
		EvalFunc: utils.Unset,
	})
}

// NewInsecureAllowAnyToken implements CIS Kubernetes v1.2.0 1.1.3
func NewInsecureAllowAnyToken() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.3",
		Description: "Ensure that the --insecure-allow-any-token argument is not set",

		Field:    "insecure-allow-any-token",
		EvalFunc: utils.Unset,
	})
}

// NewKubeletHTTPS implements CIS Kubernetes v1.2.0 1.1.4
func NewKubeletHTTPS() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.4",
		Description: "Ensure that the --kubelet-https argument is set to true",

		Field:        "kubelet-https",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewInsecureBindAddress implements CIS Kubernetes v1.2.0 1.1.5
func NewInsecureBindAddress() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.5",
		Description: "Ensure that the --insecure-bind-address argument is not set",

		Field:        "insecure-bind-address",
		Default:      "127.0.0.1",
		EvalFunc:     utils.Matches,
		DesiredValue: "127.0.0.1",
	})
}

// NewInsecurePort implements CIS Kubernetes v1.2.0 1.1.8
func NewInsecurePort() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.6",
		Description: "Ensure that the --insecure-port argument is set to 0",

		Field:        "insecure-port",
		Default:      "8080",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// 1.1.7 is uniquely dealing with integer port values

// NewProfiling implements CIS Kubernetes v1.2.0 1.1.8
func NewProfiling() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.8",
		Description: "Ensure that the --profiling argument is set to false",

		Field:        "profiling",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewRepairMalformedUpdates implements CIS Kubernetes v1.2.0 1.1.9
func NewRepairMalformedUpdates() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.9",
		Description: "Ensure that the --repair-malformed-updates argument is set to false",

		Field:        "repair-malformed-updates",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewAlwaysAdmitPolicy implements CIS Kubernetes v1.2.0 1.1.10
func NewAlwaysAdmitPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.10",
		Description: "Ensure that the admission control policy is not set to AlwaysAdmit",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AlwaysAdmit",
	})
}

// NewAlwaysPullImagesPolicy implements CIS Kubernetes v1.2.0 1.1.11
func NewAlwaysPullImagesPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.11",
		Description: "Ensure that the admission control policy is set to AlwaysPullImages",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "AlwaysPullImages",
	})
}

// NewDenyEscalatingExecPolicy implements CIS Kubernetes v1.2.0 1.1.12
func NewDenyEscalatingExecPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.12",
		Description: "Ensure that the admission control policy is set to DenyEscalatingExec",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "DenyEscalatingExec",
	})
}

// NewSecurityContextDenyPolicy implements CIS Kubernetes v1.2.0 1.1.13
func NewSecurityContextDenyPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.13",
		Description: "Ensure that the admission control policy is set to SecurityContextDeny",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "SecurityContextDeny",
	})
}

// NewNamespaceLifecyclePolicy implements CIS Kubernetes v1.2.0 1.1.14
func NewNamespaceLifecyclePolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.14",
		Description: "Ensure that the admission control policy is set to NamespaceLifecycle",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "NamespaceLifecycle",
	})
}

// NewAuditLogPath implements CIS Kubernetes v1.2.0 1.1.15
func NewAuditLogPath() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.15",
		Description: "Ensure that the --audit-log-path argument is set as appropriate",

		Field:    "audit-log-path",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAuditLogMaxAge implements CIS Kubernetes v1.2.0 1.1.16
func NewAuditLogMaxAge() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.16",
		Description: "Ensure that the --audit-log-maxage argument is set to 30 or as appropriate",

		Field:    "audit-log-maxage",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAuditLogMaxBackup implements CIS Kubernetes v1.2.0 1.1.17
func NewAuditLogMaxBackup() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.17",
		Description: "Ensure that the --audit-log-maxbackup argument is set to 10 or as appropriate",

		Field:    "audit-log-maxbackup",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAuditLogMaxSize implements CIS Kubernetes v1.2.0 1.1.18
func NewAuditLogMaxSize() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.18",
		Description: "Ensure that the --audit-log-maxsize argument is set to 100 or as appropriate",

		Field:    "audit-log-maxsize",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAlwaysAllowMode implements CIS Kubernetes v1.2.0 1.1.19
func NewAlwaysAllowMode() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.19",
		Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",

		Field:        "authorization-mode",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AlwaysAllow",
	})
}

// NewTokenAuthFile implements CIS Kubernetes v1.2.0 1.1.20
func NewTokenAuthFile() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.20",
		Description: "Ensure that the --token-auth-file parameter is not set",

		Field:    "token-auth-file",
		EvalFunc: utils.Unset,
	})
}

// NewKubeletCertificationAuthority implements CIS Kubernetes v1.2.0 1.1.21
func NewKubeletCertificationAuthority() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.21",
		Description: "Ensure that the --kubelet-certificate-authority argument is set as appropriate",

		Field:    "kubelet-certificate-authority",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewKubeletCertificationAuthory implements CIS Kubernetes v1.2.0 1.1.22
func NewKubeletCertificationAuthory() utils.Check {
	return newMultipleKubernetesAPIServerCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.22",
		Description: "Ensure that the --kubelet-client-certificate and --kubelet-client-key arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "--kubelet-client-certificate",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "--kubelet-client-key",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewServiceAccountLookup implements CIS Kubernetes v1.2.0 1.1.23
func NewServiceAccountLookup() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.23",
		Description: "Ensure that the --service-account-lookup argument is set to true",

		Field:        "service-account-lookup",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewPodSecurityPolicy implements CIS Kubernetes v1.2.0 1.1.24
func NewPodSecurityPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.24",
		Description: "Ensure that the admission control policy is set to PodSecurityPolicy",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "PodSecurityPolicy",
	})
}

// NewServiceAcountKeyFile implements CIS Kubernetes v1.2.0 1.1.25
func NewServiceAcountKeyFile() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.25",
		Description: "Ensure that the --service-account-key-file argument is set as appropriate",

		Field:    "service-account-key-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewEtcdCerts implements CIS Kubernetes v1.2.0 1.1.26
func NewEtcdCerts() utils.Check {
	return newMultipleKubernetesAPIServerCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.26",
		Description: "Ensure that the --etcd-certfile and --etcd-keyfile arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "--etcd-certfile",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "--etcd-keyfile",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewServiceAccountPolicy implements CIS Kubernetes v1.2.0 1.1.27
func NewServiceAccountPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.27",
		Description: "Ensure that the admission control policy is set to ServiceAccount",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "ServiceAccount",
	})
}

// NewTLSCerts implements CIS Kubernetes v1.2.0 1.1.28
func NewTLSCerts() utils.Check {
	return newMultipleKubernetesAPIServerCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.28",
		Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "--tls-cert-file",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "--tls-private-key-file",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewClientCAFile implements CIS Kubernetes v1.2.0 1.1.29
func NewClientCAFile() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.29",
		Description: "Ensure that the --client-ca-file argument is set as appropriate",

		Field:    "client-ca-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewEtcdCAFile implements CIS Kubernetes v1.2.0 1.1.30
func NewEtcdCAFile() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.30",
		Description: "Ensure that the --etcd-cafile argument is set as appropriate",

		Field:    "etcd-cafile",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewNodeMode implements CIS Kubernetes v1.2.0 1.1.31
func NewNodeMode() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.31",
		Description: "Ensure that the --authorization-mode argument is set to Node",

		Field:        "authorization-mode",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.Contains,
		DesiredValue: "Node",
	})
}

// NewNodeRestrictionPolicy implements CIS Kubernetes v1.2.0 1.1.32
func NewNodeRestrictionPolicy() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.32",
		Description: "Ensure that the admission control policy is set to NodeRestriction",

		Field:        "admission-control",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.Contains,
		DesiredValue: "NodeRestriction",
	})
}

// NewExperimentalEncryptionProviderConfig implements CIS Kubernetes v1.2.0 1.1.33
func NewExperimentalEncryptionProviderConfig() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.33",
		Description: "Ensure that the --experimental-encryption-provider-config argument is set as appropriate",

		Field:    "experimental-encryption-provider-config",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// v1.2.0 1.1.34 is checking for an encryption provider

// NewEventRateLimit implements CIS Kubernetes v1.2.0 1.1.35
func NewEventRateLimit() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.35",
		Description: "Ensure that the admission control policy is set to EventRateLimit",

		Field:        "admission-control",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.Contains,
		DesiredValue: "EventRateLimit",
	})
}

// NewAdvancedAuditing implements CIS Kubernetes v1.2.0 1.1.36
func NewAdvancedAuditing() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.36",
		Description: "Ensure that the AdvancedAuditing argument is not set to false",

		Field:        "feature-gates",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AdvancedAuditing=false",
	})
}

// NewRequestTimeout implements CIS Kubernetes v1.2.0 1.1.37
func NewRequestTimeout() utils.Check {
	return newKubernetesAPIServerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.1.37",
		Description: "Ensure that the --request-timeout argument is set as appropriate",

		Field:    "request-timeout",
		EvalFunc: utils.SetAsAppropriate,
	})
}

func init() {
	checks.AddToRegistry(
		NewAnonymousAuth(),
		NewBasicAuthFile(),
		NewInsecureAllowAnyToken(),
		NewKubeletHTTPS(),
		NewInsecureBindAddress(),
		NewInsecurePort(),
		NewProfiling(),
		NewRepairMalformedUpdates(),
		NewAlwaysAdmitPolicy(),
		NewAlwaysPullImagesPolicy(),
		NewDenyEscalatingExecPolicy(),
		NewSecurityContextDenyPolicy(),
		NewNamespaceLifecyclePolicy(),
		NewAuditLogPath(),
		NewAuditLogMaxAge(),
		NewAuditLogMaxBackup(),
		NewAuditLogMaxSize(),
		NewAlwaysAllowMode(),
		NewTokenAuthFile(),
		NewKubeletCertificationAuthority(),
		NewKubeletCertificationAuthory(),
		NewServiceAccountLookup(),
		NewPodSecurityPolicy(),
		NewServiceAcountKeyFile(),
		NewEtcdCerts(),
		NewServiceAccountPolicy(),
		NewTLSCerts(),
		NewClientCAFile(),
		NewEtcdCAFile(),
		NewNodeMode(),
		NewNodeRestrictionPolicy(),
		NewExperimentalEncryptionProviderConfig(),
		NewEventRateLimit(),
		NewAdvancedAuditing(),
		NewRequestTimeout(),
	)
}
