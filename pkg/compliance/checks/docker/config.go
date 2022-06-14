package docker

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		standards.CISDockerCheckName("3_1"): common.SystemdOwnershipCheck("docker.service", "root", "root"),
		standards.CISDockerCheckName("3_2"): common.SystemdPermissionCheck("docker.service", 0644),

		standards.CISDockerCheckName("3_3"): common.OptionalSystemdOwnershipCheck("docker.socket", "root", "root"),
		standards.CISDockerCheckName("3_4"): common.OptionalSystemdPermissionCheck("docker.socket", 0644),

		standards.CISDockerCheckName("3_5"): common.OwnershipCheck("/etc/docker", "root", "root"),
		standards.CISDockerCheckName("3_6"): common.PermissionCheck("/etc/docker", 0755),

		standards.CISDockerCheckName("3_7"): common.RecursiveOwnershipCheck("/etc/docker/certs.d", "root", "root"),
		standards.CISDockerCheckName("3_8"): common.RecursivePermissionCheck("/etc/docker/certs.d", 0444),

		standards.CISDockerCheckName("3_9"):  common.CommandLineFileOwnership("dockerd", "tlscacert", "root", "root"),
		standards.CISDockerCheckName("3_10"): common.CommandLineFilePermissions("dockerd", "tlscacert", 0444),

		standards.CISDockerCheckName("3_11"): common.CommandLineFileOwnership("dockerd", "tlscert", "root", "root"),
		standards.CISDockerCheckName("3_12"): common.CommandLineFilePermissions("dockerd", "tlscert", 0444),

		standards.CISDockerCheckName("3_13"): common.CommandLineFileOwnership("dockerd", "tlskey", "root", "root"),
		standards.CISDockerCheckName("3_14"): common.CommandLineFilePermissions("dockerd", "tlskey", 0400),

		standards.CISDockerCheckName("3_15"): common.OwnershipCheck("/var/run/docker.sock", "root", "docker"),
		standards.CISDockerCheckName("3_16"): common.PermissionCheck("/var/run/docker.sock", 0660),

		standards.CISDockerCheckName("3_17"): common.OptionalOwnershipCheck("/etc/docker/daemon.json", "root", "root"),
		standards.CISDockerCheckName("3_18"): common.OptionalPermissionCheck("/etc/docker/daemon.json", 0644),

		standards.CISDockerCheckName("3_19"): common.OptionalOwnershipCheck("/etc/default/docker", "root", "root"),

		standards.CISDockerCheckName("3_20"): common.OptionalOwnershipCheck("/etc/sysconfig/docker", "root", "root"),
		standards.CISDockerCheckName("3_21"): common.OptionalPermissionCheck("/etc/sysconfig/docker", 0644),

		standards.CISDockerCheckName("3_22"): common.OptionalPermissionCheck("/etc/default/docker", 0644),
	})
}
