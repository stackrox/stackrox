package etcd

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

const process = "etcd"

var configFunc = utils.GetEtcdConfig

func newEtcdCheck(check *utils.CommandCheck) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

func newMultipleEtcdCheck(check *utils.MultipleCommandChecks) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

// NewEtcdCertFiles implements CIS Kubernetes v1.2.0 1.5.1
func NewEtcdCertFiles() utils.Check {
	return newMultipleEtcdCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.1",
		Description: "Ensure that the --cert-file and --key-file arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "cert-file",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "key-file ",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewClientCertAuth implements CIS Kubernetes v1.2.0 1.5.2
func NewClientCertAuth() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.2",
		Description: "Ensure that the --client-cert-auth argument is set to true",

		Field:        "client-cert-auth",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewAutoTLS implements CIS Kubernetes v1.2.0 1.5.3
func NewAutoTLS() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.3",
		Description: "Ensure that the --auto-tls argument is not set to true",

		Field:        "auto-tls",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewPeerCertFiles implements CIS Kubernetes v1.2.0 1.5.4
func NewPeerCertFiles() utils.Check {
	return newMultipleEtcdCheck(&utils.MultipleCommandChecks{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.4",
		Description: "Ensure that the --peer-cert-file and --peer-key-file arguments are set as appropriate",
		Checks: []utils.CommandCheck{
			{
				Field:    "peer-cert-file",
				EvalFunc: utils.SetAsAppropriate,
			},
			{
				Field:    "peer-key-file ",
				EvalFunc: utils.SetAsAppropriate,
			},
		},
	})
}

// NewPeerClientCertAuth implements CIS Kubernetes v1.2.0 1.5.5
func NewPeerClientCertAuth() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.5",
		Description: "Ensure that the --peer-client-cert-auth argument is set to true",

		Field:        "peer-client-cert-auth",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewPeerAutoTLS implements CIS Kubernetes v1.2.0 1.5.6
func NewPeerAutoTLS() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.6",
		Description: "Ensure that the --peer-auto-tls argument is not set to true",

		Field:        "peer-auto-tls",
		Default:      "false",
		EvalFunc:     utils.Matches,
		DesiredValue: "true",
	})
}

// NewWalDir implements CIS Kubernetes v1.2.0 1.5.7
func NewWalDir() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.7",
		Description: "Ensure that the --wal-dir argument is set as appropriate",

		Field:    "wal-dir",
		EvalFunc: utils.SetAsAppropriate,
	})
}

// NewWalMax implements CIS Kubernetes v1.2.0 1.5.8
func NewWalMax() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.8",
		Description: "Ensure that the --max-wals argument is set to 0",

		Field:        "max-wals",
		Default:      "5",
		EvalFunc:     utils.Matches,
		DesiredValue: "0",
	})
}

// NewUniqueCA implements CIS Kubernetes v1.2.0 1.5.9
func NewUniqueCA() utils.Check {
	return newEtcdCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 1.5.9",
		Description: "Ensure that a unique Certificate Authority is used for etcd",

		EvalFunc: utils.Skip,
	})
}

func init() {
	checks.AddToRegistry(
		NewEtcdCertFiles(),
		NewClientCertAuth(),
		NewAutoTLS(),
		NewPeerCertFiles(),
		NewPeerClientCertAuth(),
		NewPeerAutoTLS(),
		NewWalDir(),
		NewWalMax(),
		NewUniqueCA(),
	)
}
