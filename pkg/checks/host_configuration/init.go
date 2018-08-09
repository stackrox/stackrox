package hostconfiguration

import "github.com/stackrox/rox/pkg/checks"

func init() {
	checks.AddToRegistry(
		NewContainerPartitionBenchmark(), // 1.1
		NewHostHardened(),
		NewDockerUpdated(),
		NewTrustedUsers(),
		NewDockerDaemonAudit(), // 1.5
		NewVarLibDockerAudit(),
		NewEtcDockerAudit(),
		NewDockerServiceAudit(),
		NewDockerSocketAudit(),
		NewEtcDefaultDockerAudit(), // 1.10
		NewEtcDockerDaemonJSONAudit(),
		NewUsrBinContainerdAudit(),
		NewUsrBinRundAudit(),
	)
}
