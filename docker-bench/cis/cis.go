package cis

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/docker-bench/cis/configuration_files"
	"bitbucket.org/stack-rox/apollo/docker-bench/cis/container_images_and_build"
	"bitbucket.org/stack-rox/apollo/docker-bench/cis/container_runtime"
	"bitbucket.org/stack-rox/apollo/docker-bench/cis/docker_daemon_configuration"
	"bitbucket.org/stack-rox/apollo/docker-bench/cis/docker_security_operations"
	"bitbucket.org/stack-rox/apollo/docker-bench/cis/host_configuration"
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

var cisBenchmarks = []utils.Check{
	// Part 1
	hostconfiguration.NewContainerPartitionBenchmark(), // 1.1
	hostconfiguration.NewHostHardened(),
	hostconfiguration.NewDockerUpdated(),
	hostconfiguration.NewTrustedUsers(),
	hostconfiguration.NewDockerDaemonAudit(), // 1.5
	hostconfiguration.NewVarLibDockerAudit(),
	hostconfiguration.NewEtcDockerAudit(),
	hostconfiguration.NewDockerServiceAudit(),
	hostconfiguration.NewDockerSocketAudit(),
	hostconfiguration.NewEtcDefaultDockerAudit(), // 1.10
	hostconfiguration.NewEtcDockerDaemonJSONAudit(),
	hostconfiguration.NewUsrBinContainerdAudit(),
	hostconfiguration.NewUsrBinRundAudit(),

	// Part 2
	dockerdaemonconfiguration.NewNetworkRestrictionBenchmark(), // 2.1
	dockerdaemonconfiguration.NewLogLevelBenchmark(),
	dockerdaemonconfiguration.NewIPTablesEnabledBenchmark(),
	dockerdaemonconfiguration.NewInsecureRegistriesBenchmark(),
	dockerdaemonconfiguration.NewAUFSBenchmark(), // 2.5
	dockerdaemonconfiguration.NewTLSVerifyBenchmark(),
	dockerdaemonconfiguration.NewDefaultUlimitBenchmark(),
	dockerdaemonconfiguration.NewUserNamespaceBenchmark(),
	dockerdaemonconfiguration.NewCgroupUsageBenchmark(),
	dockerdaemonconfiguration.NewBaseDeviceSizeBenchmark(), // 2.10
	dockerdaemonconfiguration.NewAuthorizationPluginBenchmark(),
	dockerdaemonconfiguration.NewRemoteLoggingBenchmark(),
	dockerdaemonconfiguration.NewDisableLegacyRegistryBenchmark(),
	dockerdaemonconfiguration.NewLiveRestoreEnabledBenchmark(),
	dockerdaemonconfiguration.NewDisableUserlandProxyBenchmark(), // 2.15
	dockerdaemonconfiguration.NewDaemonWideSeccompBenchmark(),
	dockerdaemonconfiguration.NewDisableExperimentalBenchmark(),
	dockerdaemonconfiguration.NewRestrictContainerPrivilegesBenchmark(),

	// Part 3
	configurationfiles.NewDockerServiceOwnershipCheck(), // 3.1
	configurationfiles.NewDockerServicePermissionsCheck(),
	configurationfiles.NewDockerSocketOwnershipCheck(),
	configurationfiles.NewDockerSocketPermissionsCheck(),
	configurationfiles.NewEtcDockerOwnershipCheck(), // 3.5
	configurationfiles.NewEtcDockerPermissionsCheck(),
	configurationfiles.NewRegistryCertificateOwnershipCheck(),
	configurationfiles.NewRegistryCertificatePermissionsCheck(),
	configurationfiles.NewTLSCACertificateOwnershipCheck(),
	configurationfiles.NewTLSCACertificatePermissionsCheck(), // 3.10
	configurationfiles.NewDockerServerCertificateOwnershipCheck(),
	configurationfiles.NewDockerServerCertificatePermissionsCheck(),
	configurationfiles.NewDockerServerCertificateKeyFileOwnershipCheck(),
	configurationfiles.NewDockerServerCertificateKeyFilePermissionsCheck(),
	configurationfiles.NewDockerSocketFileOwnershipCheck(), // 3.15
	configurationfiles.NewDockerSocketFilePermissionsCheck(),
	configurationfiles.NewEtcDaemonJSONFileOwnershipCheck(),
	configurationfiles.NewEtcDaemonJSONPermissionsCheck(),
	configurationfiles.NewEtcDefaultDockerFileOwnershipCheck(),
	configurationfiles.NewEtcDefaultDockerPermissionsCheck(), // 3.20

	// Part 4
	containerimagesandbuild.NewContainerUserBenchmark(),
	containerimagesandbuild.NewTrustedBaseImagesBenchmark(),
	containerimagesandbuild.NewUnnecessaryPackagesBenchmark(),
	containerimagesandbuild.NewScannedImagesBenchmark(),
	containerimagesandbuild.NewContentTrustBenchmark(),
	containerimagesandbuild.NewImageHealthcheckBenchmark(),
	containerimagesandbuild.NewImageUpdateInstructionsBenchmark(),
	containerimagesandbuild.NewSetuidSetGidPermissionsBenchmark(),
	containerimagesandbuild.NewImageCopyBenchmark(),
	containerimagesandbuild.NewImageSecretsBenchmark(),
	containerimagesandbuild.NewVerifiedPackagesBenchmark(),

	// Part 5
	containerruntime.NewAppArmorBenchmark(), // 5.1
	containerruntime.NewSELinuxBenchmark(),
	containerruntime.NewCapabilitiesBenchmark(),
	containerruntime.NewPrivilegedBenchmark(),
	containerruntime.NewSensitiveHostMountsBenchmark(), // 5.5
	containerruntime.NewSSHBenchmark(),
	containerruntime.NewPrivilegedPortsBenchmark(),
	containerruntime.NewNecessaryPortsBenchmark(),
	containerruntime.NewSharedNetworkBenchmark(),
	containerruntime.NewMemoryBenchmark(), // 5.10
	containerruntime.NewCPUPriorityBenchmark(),
	containerruntime.NewReadonlyRootfsBenchmark(),
	containerruntime.NewSpecificHostInterfaceBenchmark(),
	containerruntime.NewRestartPolicyBenchmark(),
	containerruntime.NewPidNamespaceBenchmark(), // 5.15
	containerruntime.NewIpcNamespaceBenchmark(),
	containerruntime.NewHostDevicesBenchmark(),
	containerruntime.NewUlimitBenchmark(),
	containerruntime.NewMountPropagationBenchmark(),
	containerruntime.NewUTSNamespaceBenchmark(), // 5.20
	containerruntime.NewSeccompBenchmark(),
	containerruntime.NewPrivilegedDockerExecBenchmark(),
	containerruntime.NewUserDockerExecBenchmark(),
	containerruntime.NewCgroupBenchmark(),
	containerruntime.NewAcquiringPrivilegesBenchmark(), // 5.25
	containerruntime.NewRuntimeHealthcheckBenchmark(),
	containerruntime.NewLatestImageBenchmark(),
	containerruntime.NewPidCgroupBenchmark(),
	containerruntime.NewBridgeNetworkBenchmark(),
	containerruntime.NewUsernsBenchmark(), // 5.30
	containerruntime.NewDockerSocketMountBenchmark(),

	// Part 6
	dockersecurityoperations.NewImageSprawlBenchmark(),
	dockersecurityoperations.NewContainerSprawlBenchmark(),
}

// RunCISBenchmark runs the CIS benchmark
func RunCISBenchmark() []*v1.CheckResult {
	results := make([]*v1.CheckResult, 0, len(cisBenchmarks))
	for _, benchmark := range cisBenchmarks {
		for _, dep := range benchmark.Definition().Dependencies {
			if err := dep(); err != nil {
				fmt.Printf("Skipping Test %v due to err in dependency: %+v\n", benchmark.Definition().Name, err)
				continue
			}
		}
		result := benchmark.Run()

		definition := benchmark.Definition().CheckDefinition
		result.Definition = &definition
		results = append(results, &result)
	}
	return results
}
