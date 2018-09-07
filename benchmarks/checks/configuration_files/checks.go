package configurationfiles

import (
	"os"

	"github.com/stackrox/rox/benchmarks/checks/utils"
)

// NewDockerServiceOwnershipCheck implements CIS-3.1
func NewDockerServiceOwnershipCheck() utils.Check {
	return utils.NewSystemdOwnershipCheck(
		"CIS Docker v1.1.0 - 3.1",
		"Ensure that docker.service file ownership is set to root:root",
		"docker.service",
		"root",
		"root",
	)
}

// NewDockerServicePermissionsCheck implements CIS-3.2
func NewDockerServicePermissionsCheck() utils.Check {
	return utils.NewSystemdPermissionsCheck(
		"CIS Docker v1.1.0 - 3.2",
		"Ensure that docker.service file permissions are set to 644 or more restrictive",
		"docker.service",
		0644,
		true,
	)
}

// NewDockerSocketOwnershipCheck implements CIS-3.3
func NewDockerSocketOwnershipCheck() utils.Check {
	return utils.NewSystemdOwnershipCheck(
		"CIS Docker v1.1.0 - 3.3",
		"Ensure that docker.socket file ownership is set to root:root",
		"docker.socket",
		"root",
		"root",
	)
}

// NewDockerSocketPermissionsCheck implements CIS-3.4
func NewDockerSocketPermissionsCheck() utils.Check {
	return utils.NewSystemdPermissionsCheck(
		"CIS Docker v1.1.0 - 3.4",
		"Ensure that docker.socket file permissions are set to 644 or more restrictive",
		"docker.socket",
		0644,
		true,
	)
}

// NewEtcDockerOwnershipCheck implements CIS-3.5
func NewEtcDockerOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.5",
		"Ensure that /etc/docker file ownership is set to root:root",
		"/etc/docker",
		"root",
		"root",
	)
}

// NewEtcDockerPermissionsCheck implements CIS-3.6
func NewEtcDockerPermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.6",
		"Ensure that /etc/docker directory permissions are set to 755 or more restrictive",
		"/etc/docker",
		0755,
		true,
	)
}

// NewRegistryCertificateOwnershipCheck implements CIS-3.7
func NewRegistryCertificateOwnershipCheck() utils.Check {
	return utils.NewRecursiveOwnershipCheck(
		"CIS Docker v1.1.0 - 3.7",
		"Ensure that registry certificate file ownership is set to root:root",
		"/etc/docker/certs.d",
		"root",
		"root",
	)
}

// NewRegistryCertificatePermissionsCheck implements CIS-3.8
func NewRegistryCertificatePermissionsCheck() utils.Check {
	return utils.NewRecursivePermissionsCheck(
		"CIS Docker v1.1.0 - 3.8",
		"Ensure that registry certificate file permissions are set to 444 or more restrictive",
		"/etc/docker/certs.d",
		0444,
		true,
	)
}

// NewTLSCACertificateOwnershipCheck implements CIS-3.9
func NewTLSCACertificateOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.9",
		"Ensure that TLS CA certificate file ownership is set to root:root",
		os.Getenv("TLS_CA_CERTIFICATE_FILE"),
		"root",
		"root",
	)
}

// NewTLSCACertificatePermissionsCheck implements CIS-3.10
func NewTLSCACertificatePermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.10",
		"Ensure that TLS CA certificate file permissions are set to 444 or more restrictive",
		os.Getenv("TLS_CA_CERTIFICATE_FILE"),
		0444,
		true,
	)
}

// NewDockerServerCertificateOwnershipCheck implements CIS-3.11
func NewDockerServerCertificateOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.11",
		"Ensure that Docker server certificate file ownership is set to root:root",
		os.Getenv("DOCKER_SERVER_CERTIFICATE_FILE"),
		"root",
		"root",
	)
}

// NewDockerServerCertificatePermissionsCheck implements CIS-3.12
func NewDockerServerCertificatePermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.12",
		"Ensure that Docker server certificate file permissions are set to 444 or more restrictive",
		os.Getenv("DOCKER_SERVER_CERTIFICATE_FILE"),
		0444,
		true,
	)
}

// NewDockerServerCertificateKeyFileOwnershipCheck implements CIS-3.13
func NewDockerServerCertificateKeyFileOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.13",
		"Ensure that Docker server certificate key file ownership is set to root:root",
		os.Getenv("DOCKER_SERVER_CERTIFICATE_KEY_FILE"),
		"root",
		"root",
	)
}

// NewDockerServerCertificateKeyFilePermissionsCheck implements CIS-3.14
func NewDockerServerCertificateKeyFilePermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.14",
		"Ensure that Docker server certificate key file permissions are set to 400",
		os.Getenv("DOCKER_SERVER_CERTIFICATE_KEY_FILE"),
		0400,
		true,
	)
}

// NewDockerSocketFileOwnershipCheck implements CIS-3.15
func NewDockerSocketFileOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.15",
		"Ensure that Docker socket file ownership is set to root:docker",
		"/var/run/docker.sock",
		"root",
		"docker",
	)
}

// NewDockerSocketFilePermissionsCheck implements CIS-3.16
func NewDockerSocketFilePermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.16",
		"Ensure that Docker socket file permissions are set to 660 or more restrictive",
		"/var/run/docker.sock",
		0660,
		true,
	)
}

// NewEtcDaemonJSONFileOwnershipCheck implements CIS-3.17
func NewEtcDaemonJSONFileOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.17",
		"Ensure that daemon.json file ownership is set to root:root",
		"/etc/docker/daemon.json",
		"root",
		"root",
	)
}

// NewEtcDaemonJSONPermissionsCheck implements CIS-3.18
func NewEtcDaemonJSONPermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.18",
		"Ensure that daemon.json file permissions are set to 644 or more restrictive",
		"/etc/docker/daemon.json",
		0644,
		true,
	)
}

// NewEtcDefaultDockerFileOwnershipCheck implements CIS-3.19
func NewEtcDefaultDockerFileOwnershipCheck() utils.Check {
	return utils.NewOwnershipCheck(
		"CIS Docker v1.1.0 - 3.19",
		"Ensure that /etc/default/docker file ownership is set to root:root",
		"/etc/default/docker",
		"root",
		"root",
	)
}

// NewEtcDefaultDockerPermissionsCheck implements CIS-3.20
func NewEtcDefaultDockerPermissionsCheck() utils.Check {
	return utils.NewPermissionsCheck(
		"CIS Docker v1.1.0 - 3.20",
		"Ensure that /etc/default/docker file permissions are set to 644 or more restrictive",
		"/etc/default/docker",
		0644,
		true,
	)
}
