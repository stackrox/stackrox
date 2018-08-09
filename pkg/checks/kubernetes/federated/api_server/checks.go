package federationserver

import (
	"github.com/stackrox/rox/pkg/checks"
	"github.com/stackrox/rox/pkg/checks/utils"
)

const process = "federation-apiserver"

var configFunc = utils.GetKubeFederationAPIServerConfig

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

// NewAnonymousAuth implements CIS Kubernetes v1.2.0 3.1.1
func NewAnonymousAuth() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.1",
		Description: "Ensure that the --anonymous-auth argument is set to false",

		Field:        "anonymous-auth",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewBasicAuthFile implements CIS Kubernetes v1.2.0 3.1.2
func NewBasicAuthFile() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.2",
		Description: "Ensure that the --basic-auth-file argument is not set",

		Field:    "basic-auth-file",
		EvalFunc: utils.Unset,
	})
}

// NewInsecureAllowAnyToken implements CIS Kubernetes v1.2.0 3.1.3
func NewInsecureAllowAnyToken() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.3",
		Description: "Ensure that the --insecure-allow-any-token argument is not set",

		Field:    "insecure-allow-any-token",
		EvalFunc: utils.Unset,
	})
}

// NewInsecureBindAddress implements CIS Kubernetes v1.2.0 3.1.4
func NewInsecureBindAddress() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.4",
		Description: "Ensure that the --insecure-bind-address argument is not set",

		Field:    "insecure-bind-address",
		EvalFunc: utils.Unset,
	})
}

// NewInsecurePort implements CIS Kubernetes v1.2.0 3.1.5
func NewInsecurePort() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.5",
		Description: "Ensure that the --insecure-port argument is set to 0",

		Field:        "insecure-port",
		Default:      "8080",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// NewSecurePort implements CIS Kubernetes v1.2.0 3.1.6
func NewSecurePort() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.6",
		Description: "Ensure that the --secure-port argument is not set to 0",

		Field:        "secure-port",
		Default:      "6443",
		EvalFunc:     utils.NotMatches,
		DesiredValue: "0",
	})
}

// NewProfiling implements CIS Kubernetes v1.2.0 3.1.7
func NewProfiling() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.7",
		Description: "Ensure that the --profiling argument is set to false",

		Field:        "profiling",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

// NewAlwaysAdmitPolicy implements CIS Kubernetes v1.2.0 3.1.8
func NewAlwaysAdmitPolicy() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.8",
		Description: "Ensure that the admission control policy is not set to AlwaysAdmit",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AlwaysAdmit",
	})
}

// NewNamespaceLifecyclePolicy implements CIS Kubernetes v1.2.0 3.1.9
func NewNamespaceLifecyclePolicy() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.9",
		Description: "Ensure that the admission control policy is set to NamespaceLifecycle",

		Field:        "admission-control",
		Default:      "AlwaysAdmit",
		EvalFunc:     utils.Contains,
		DesiredValue: "NamespaceLifecycle",
	})
}

// NewAuditLogPath implements CIS Kubernetes v1.2.0 3.1.10
func NewAuditLogPath() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.10",
		Description: "Ensure that the --audit-log-path argument is set as appropriate",

		Field:    "audit-log-path",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewMaxAge implements CIS Kubernetes v1.2.0 3.1.11
func NewMaxAge() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.11",
		Description: "Ensure that the --audit-log-maxage argument is set to 30 or as appropriate",

		Field:    "audit-log-maxage",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAuditLogMaxBackup implements CIS Kubernetes v1.2.0 3.1.12
func NewAuditLogMaxBackup() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.12",
		Description: "Ensure that the --audit-log-maxbackup argument is set to 10 or as appropriate",

		Field:    "audit-log-maxbackup",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAuditLogMaxSize implements CIS Kubernetes v1.2.0 3.1.13
func NewAuditLogMaxSize() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.13",
		Description: "Ensure that the --audit-log-maxsize argument is set to 100 or as appropriate",

		Field:    "audit-log-maxsize",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewAlwaysAllowMode implements CIS Kubernetes v1.2.0 3.1.14
func NewAlwaysAllowMode() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.14",
		Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",

		Field:        "authorization-mode",
		Default:      "AlwaysAllow",
		EvalFunc:     utils.NotContains,
		DesiredValue: "AlwaysAllow",
	})
}

// NewTokenAuthFile implements CIS Kubernetes v1.2.0 3.1.15
func NewTokenAuthFile() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.15",
		Description: "Ensure that the --token-auth-file parameter is not set",

		Field:    "token-auth-file",
		EvalFunc: utils.Unset,
	})
}

// NewServiceAccountLookup implements CIS Kubernetes v1.2.0 3.1.16
func NewServiceAccountLookup() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.16",
		Description: "Ensure that the --service-account-lookup argument is set to true",

		Field:        "service-account-lookup",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewServiceAccountKeyFile implements CIS Kubernetes v1.2.0 3.1.17
func NewServiceAccountKeyFile() utils.Check {
	return newKubeletCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.17",
		Description: "Ensure that the --service-account-key-file argument is set as appropriate",

		Field:    "service-account-key-file",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewEtcdCertFiles implements CIS Kubernetes v1.2.0 3.1.18
func NewEtcdCertFiles() utils.Check {
	return newMultipleKubeletCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.18",
		Description: "Ensure that the --etcd-certfile and --etcd-keyfile arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "etcd-certfile",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "etcd-keyfile",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewTLSCertFiles implements CIS Kubernetes v1.2.0 3.1.19
func NewTLSCertFiles() utils.Check {
	return newMultipleKubeletCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 3.1.19",
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

func init() {
	checks.AddToRegistry(NewAnonymousAuth(),
		NewBasicAuthFile(),
		NewInsecureAllowAnyToken(),
		NewInsecureBindAddress(),
		NewInsecurePort(),
		NewSecurePort(),
		NewProfiling(),
		NewAlwaysAdmitPolicy(),
		NewNamespaceLifecyclePolicy(),
		NewAuditLogPath(),
		NewMaxAge(),
		NewAuditLogMaxBackup(),
		NewAuditLogMaxSize(),
		NewAlwaysAllowMode(),
		NewTokenAuthFile(),
		NewServiceAccountLookup(),
		NewServiceAccountKeyFile(),
		NewEtcdCertFiles(),
		NewTLSCertFiles(),
	)
}
