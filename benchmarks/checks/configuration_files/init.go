package configurationfiles

import "github.com/stackrox/rox/benchmarks/checks"

func init() {
	checks.AddToRegistry( // Part 3
		NewDockerServiceOwnershipCheck(), // 3.1
		NewDockerServicePermissionsCheck(),
		NewDockerSocketOwnershipCheck(),
		NewDockerSocketPermissionsCheck(),
		NewEtcDockerOwnershipCheck(), // 3.5
		NewEtcDockerPermissionsCheck(),
		NewRegistryCertificateOwnershipCheck(),
		NewRegistryCertificatePermissionsCheck(),
		NewTLSCACertificateOwnershipCheck(),
		NewTLSCACertificatePermissionsCheck(), // 3.10
		NewDockerServerCertificateOwnershipCheck(),
		NewDockerServerCertificatePermissionsCheck(),
		NewDockerServerCertificateKeyFileOwnershipCheck(),
		NewDockerServerCertificateKeyFilePermissionsCheck(),
		NewDockerSocketFileOwnershipCheck(), // 3.15
		NewDockerSocketFilePermissionsCheck(),
		NewEtcDaemonJSONFileOwnershipCheck(),
		NewEtcDaemonJSONPermissionsCheck(),
		NewEtcDefaultDockerFileOwnershipCheck(),
		NewEtcDefaultDockerPermissionsCheck(), // 3.20
	)
}
