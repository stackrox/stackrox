package swarm

import "bitbucket.org/stack-rox/apollo/pkg/checks"

func init() {
	checks.AddToRegistry(
		// Part 7
		NewSwarmEnabledCheck(), // 7.1
		NewMinimumManagersCheck(),
		NewHostInterfaceBind(),
		NewEncryptedNetworks(),
		NewSecretManagement(), // 7.5
		NewAutoLockCheck(),
		NewAutoLockRotationCheck(),
		NewNodeCertificates(),
		NewCACertificates(),
		NewManagementPlaneCheck(), // 7.10
	)
}
