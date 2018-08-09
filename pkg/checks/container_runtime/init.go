package containerruntime

import "github.com/stackrox/rox/pkg/checks"

func init() {
	checks.AddToRegistry(
		// Part 5
		NewAppArmorBenchmark(), // 5.1
		NewSELinuxBenchmark(),
		NewCapabilitiesBenchmark(),
		NewPrivilegedBenchmark(),
		NewSensitiveHostMountsBenchmark(), // 5.5
		NewSSHBenchmark(),
		NewPrivilegedPortsBenchmark(),
		NewNecessaryPortsBenchmark(),
		NewSharedNetworkBenchmark(),
		NewMemoryBenchmark(), // 5.10
		NewCPUPriorityBenchmark(),
		NewReadonlyRootfsBenchmark(),
		NewSpecificHostInterfaceBenchmark(),
		NewRestartPolicyBenchmark(),
		NewPidNamespaceBenchmark(), // 5.15
		NewIpcNamespaceBenchmark(),
		NewHostDevicesBenchmark(),
		NewUlimitBenchmark(),
		NewMountPropagationBenchmark(),
		NewUTSNamespaceBenchmark(), // 5.20
		NewSeccompBenchmark(),
		NewPrivilegedDockerExecBenchmark(),
		NewUserDockerExecBenchmark(),
		NewCgroupBenchmark(),
		NewAcquiringPrivilegesBenchmark(), // 5.25
		NewRuntimeHealthcheckBenchmark(),
		NewLatestImageBenchmark(),
		NewPidCgroupBenchmark(),
		NewBridgeNetworkBenchmark(),
		NewUsernsBenchmark(), // 5.30
		NewDockerSocketMountBenchmark(),
	)
}
