package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/compliance/msgfmt"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		standards.CISDockerCheckName("5_1"):  runningContainerCheck(appArmor, "has an AppArmor profile configured"),
		standards.CISDockerCheckName("5_2"):  runningContainerCheck(selinux, "has SELinux configured"),
		standards.CISDockerCheckName("5_3"):  runningContainerCheck(capabilities, "has extra capabilities enabled"),
		standards.CISDockerCheckName("5_4"):  runningContainerCheck(privileged, "is not running in privileged mode"),
		standards.CISDockerCheckName("5_5"):  runningContainerCheck(sensitiveHostMounts, "does not mount any sensitive host directories"),
		standards.CISDockerCheckName("5_7"):  runningContainerCheck(privilegedPorts, "does not bind to a privileged host port"),
		standards.CISDockerCheckName("5_8"):  runningContainerCheck(necessaryPorts, "does not bind to any host ports"),
		standards.CISDockerCheckName("5_9"):  runningContainerCheck(sharedNetwork, "does not use the 'host' network mode"),
		standards.CISDockerCheckName("5_10"): runningContainerCheck(memoryLimit, "has memory limits configured"),
		standards.CISDockerCheckName("5_11"): runningContainerCheck(cpuShares, "has CPU shares configured"),
		standards.CISDockerCheckName("5_12"): runningContainerCheck(readonlyFS, "uses a read-only root filesystem"),
		standards.CISDockerCheckName("5_13"): runningContainerCheck(specificHostInterface, "does not bind to all host interface addresses (0.0.0.0)"),
		standards.CISDockerCheckName("5_14"): runningContainerCheck(restartPolicy, "has an on-failure restart policy with a maximum of 5 retries"),
		standards.CISDockerCheckName("5_15"): runningContainerCheck(pidNamespace, "is not using the host PID namespace"),
		standards.CISDockerCheckName("5_16"): runningContainerCheck(ipcNamespace, "is not using the host IPC namespace"),
		standards.CISDockerCheckName("5_17"): runningContainerCheck(hostDevices, "has not mounted any host devices"),
		standards.CISDockerCheckName("5_18"): runningContainerCheck(ulimit, "does not override ulimits"),
		standards.CISDockerCheckName("5_19"): runningContainerCheck(mountPropagation, "does not have any mounts that use shared propagation"),
		standards.CISDockerCheckName("5_20"): runningContainerCheck(utsNamespace, "does not use the host UTS namespace"),
		standards.CISDockerCheckName("5_21"): runningContainerCheck(seccomp, "does not have seccomp set to unconfined"),

		// 5.22 and 5.23 are in file.go

		standards.CISDockerCheckName("5_24"): runningContainerCheck(cgroup, "does not use a non-standard cgroup parent"),
		standards.CISDockerCheckName("5_25"): runningContainerCheck(acquiringPrivileges, "sets no-new-privileges in its security options"),
		standards.CISDockerCheckName("5_26"): runningContainerCheck(healthcheck, "has a correctly configured health check"),
		standards.CISDockerCheckName("5_27"): common.NoteCheck("Pulling images is invasive and not always possible depending on credential management"),
		standards.CISDockerCheckName("5_28"): runningContainerCheck(pidCgroup, "has a PID limit set"),
		standards.CISDockerCheckName("5_29"): runningContainerCheck(bridgeNetwork, "is not running on the bridge network"),
		standards.CISDockerCheckName("5_30"): runningContainerCheck(userNamespace, "is not using the host user namespace"),
		standards.CISDockerCheckName("5_31"): runningContainerCheck(noDockerSocket, "does not mount the docker socket"),

		// One off
		standards.CISDockerCheckName("4_1"): runningContainerCheck(usersInContainer, "is not running as the root user"),
	})
}

// Removed runningOnly parameter because it was always true
func runningContainerCheck(f func(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence, desc string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: common.CheckWithDockerData(func(data *types.Data) []*storage.ComplianceResultValue_Evidence {
			var results []*storage.ComplianceResultValue_Evidence
			for _, c := range data.Containers {
				if c.State == nil || !c.State.Running {
					continue
				}
				if c.HostConfig == nil {
					continue
				}
				results = append(results, f(c)...)
			}
			return results
		}),
		Metadata: &standards.Metadata{
			InterpretationText: fmt.Sprintf("StackRox checks that every running container on each node %s", desc),
			TargetKind:         framework.NodeKind,
		},
	}
}

func acquiringPrivileges(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var pass bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, o := range container.HostConfig.SecurityOpt {
		if strings.Contains(o, "no-new-privileges") {
			pass = true
			results = append(results, common.Passf("Container %q sets no-new-privileges", container.Name))
		}
	}
	if !pass {
		results = append(results, common.Failf("Container %q does not set no-new-privileges in security opts", container.Name))
	}
	return results
}

func appArmor(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.AppArmorProfile == "unconfined" {
		return common.FailListf("Container %q has app armor configured as unconfined", container.Name)
	}
	return common.PassListf("Container %q has app armor profile configured as %q", container.Name, container.AppArmorProfile)
}

func specificHostInterface(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.NetworkSettings == nil {
		return common.PassList("Container %q has no values set for network settings")
	}

	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			if strings.Contains(binding.HostIP, "0.0.0.0") {
				failed = true
				results = append(results, common.Failf("Container %q binds port %d to '0.0.0.0 %s'", container.Name, containerPort.Int(), binding.HostPort))
			}
		}
	}
	if !failed {
		results = append(results, common.Passf("Container %q binds no ports to all interfaces (0.0.0.0)", container.Name))
	}
	return results
}

func bridgeNetwork(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.NetworkSettings == nil {
		return common.PassListf("Container %q has no network settings", container.Name)
	}
	if _, ok := container.NetworkSettings.Networks["bridge"]; ok {
		return common.FailListf("Container %q is running on the bridge network", container.Name)
	}
	return common.PassListf("Container %q is not running on the bridge network", container.Name)
}

func capabilities(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if len(container.HostConfig.CapAdd) > 0 {
		return common.NoteListf("Container %q adds capabilities: %s", container.Name, strings.Join(container.HostConfig.CapAdd, ", "))
	}
	return common.PassListf("Container %q does not add any capabilities", container.Name)
}

func cgroup(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.CgroupParent == "docker" ||
		container.HostConfig.CgroupParent == "" || strings.Contains(container.HostConfig.CgroupParent, "kube") {
		return common.PassListf("Container %q has the cgroup parent set to %q", container.Name, container.HostConfig.CgroupParent)
	}
	return common.NoteListf("Container %q has the cgroup parent set to %q", container.Name, container.HostConfig.CgroupParent)
}

func cpuShares(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.CPUShares == 0 {
		return common.FailListf("Container %q does not have CPU shares set", container.Name)
	}
	return common.PassListf("Container %q has CPU shares set to %d", container.Name, container.HostConfig.CPUShares)
}

func healthcheck(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.State.Health == nil {
		return common.FailListf("Container %q does not have health configured", container.Name)
	}
	if container.State.Health.Status == "" {
		return common.FailListf("Container %q is currently reporting empty health", container.Name)
	}
	return common.PassListf("Container %q has health configured with status %q", container.Name, container.State.Health.Status)
}

func hostDevices(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if len(container.HostConfig.Devices) > 0 {
		devices := make([]string, 0, len(container.HostConfig.Devices))
		for _, device := range container.HostConfig.Devices {
			devices = append(devices, fmt.Sprintf("%v:%v", device.PathOnHost, device.PathInContainer))
		}
		return common.FailListf("Container %q has host devices [ %s ] exposed to it", container.Name, strings.Join(devices, " | "))
	}
	return common.PassListf("Container %q has not mounted any host devices", container.Name)
}

func ipcNamespace(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.IpcMode.IsHost() {
		return common.FailListf("Container %q has IPC mode set to 'host'", container.Name)
	}
	return common.PassListf("Container %q has IPC mode set to %q", container.Name, container.HostConfig.IpcMode)
}

func memoryLimit(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.Memory == 0 {
		return common.FailListf("Container %q does not have a memory limit", container.Name)
	}
	return common.PassListf("Container %q has a memory limit set to %d", container.Name, container.HostConfig.Memory)
}

func mountPropagation(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, containerMount := range container.Mounts {
		if containerMount.Propagation == mount.PropagationShared {
			failed = true
			results = append(results, common.Failf("Container %q has mount %q which uses shared propagation", container.Name, containerMount.Name))
		}
	}
	if !failed {
		results = append(results, common.Passf("Container %q has no mounts that use shared propagation", container.Name))
	}
	return results
}

func necessaryPorts(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.NetworkSettings == nil {
		return common.PassListf("Container %q does not have any network settings", container.Name)
	}
	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			failed = true
			results = append(results, common.Notef("Container %q binds container port '%d' to host port %q", container.Name, containerPort.Int(), binding.HostPort))
		}
	}
	if !failed {
		results = append(results, common.Passf("Container %q does not bind any container ports to host ports", container.Name))
	}
	return results
}

// Docker socket
func noDockerSocket(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, containerMount := range container.Mounts {
		if strings.Contains(containerMount.Source, "docker.sock") {
			failed = true
			results = append(results, common.Failf("Container %q has mounted docker.sock", container.Name))
		}
	}
	if !failed {
		results = append(results, common.Passf("Container %q has not mounted docker.sock", container.Name))
	}
	return results
}

func pidNamespace(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.PidMode.IsHost() {
		return common.FailListf("Container %q has PID mode set to 'host'", container.Name)
	}
	return common.PassListf("Container %q has PID mode set to %q", container.Name, container.HostConfig.PidMode)
}

func pidCgroup(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.PidsLimit <= 0 {
		return common.FailListf("Container %q does not have PIDs limit set", container.Name)
	}
	return common.PassListf("Container %q has PIDs limit set to %d", container.Name, container.HostConfig.PidsLimit)
}

func privileged(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.Privileged {
		return common.FailListf("Container %q is running as privileged", container.Name)
	}
	return common.PassListf("Container %q is not running as privileged", container.Name)
}

func privilegedPorts(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			portNum, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				failed = true
				results = append(results, common.Failf("Could not parse host port for container %q", container.Name))
			} else if portNum < 1024 {
				failed = true
				results = append(results, common.Failf("Container %q binds port '%d' to privileged host port '%d'", container.Name, containerPort.Int(), portNum))
			}
		}
	}
	if !failed {
		results = append(results, common.Passf("Container %q does not bind any ports with numbers < 1024", container.Name))
	}
	return results
}

func readonlyFS(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if !container.HostConfig.ReadonlyRootfs {
		return common.FailListf("Container %q does not have a readonly rootFS", container.Name)
	}
	return common.PassListf("Container %q had a readonly rootFS", container.Name)
}

func restartPolicy(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.RestartPolicy.Name == "always" {
		return common.FailListf("Container %q has a restart policy %q", container.Name, container.HostConfig.RestartPolicy.Name)
	}
	if container.HostConfig.RestartPolicy.Name == "" || container.HostConfig.RestartPolicy.Name == "no" {
		return common.PassListf("Container %q has no restart policy or restart policy 'no'", container.Name)
	}
	if container.HostConfig.RestartPolicy.Name == "on-failure" && container.HostConfig.RestartPolicy.MaximumRetryCount <= 5 {
		return common.PassListf("Container %q has a restart policy %q with max retries '%d'", container.Name, container.HostConfig.RestartPolicy.Name, container.HostConfig.RestartPolicy.MaximumRetryCount)
	}
	return common.FailListf("Container %q has a restart policy %q with max retries '%d'", container.Name, container.HostConfig.RestartPolicy.Name, container.HostConfig.RestartPolicy.MaximumRetryCount)
}

func seccomp(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var fail bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, opt := range container.HostConfig.SecurityOpt {
		if strings.EqualFold(opt, "seccomp:unconfined") {
			fail = true
			results = append(results, common.Failf("Container %q has seccomp set to unconfined", container.Name))
		}
	}
	if !fail {
		results = append(results, common.Passf("Container %q does not have seccomp set to unconfined", container.Name))
	}
	return results
}

func selinux(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var pass bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, opt := range container.HostConfig.SecurityOpt {
		if strings.Contains(opt, "selinux") {
			pass = true
			results = append(results, common.Passf("Container %q has selinux configured as %q", container.Name, opt))
		}
	}
	if !pass {
		results = append(results, common.Failf("Container %q does not have selinux configured", container.Name))
	}
	return results
}

var sensitiveMounts = set.NewFrozenStringSet(
	"/",
	"/boot",
	"/dev",
	"/etc",
	"/lib",
	"/proc",
	"/sys",
	"/usr",
)

func sensitiveHostMounts(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	var fail bool
	var results []*storage.ComplianceResultValue_Evidence
	for _, mount := range container.Mounts {
		if sensitiveMounts.Contains(mount.Source) {
			fail = true
			results = append(results, common.Failf("Container %q has sensitive mount %q with mode %q", container.Name, mount.Source, mount.Mode))
		}
	}
	if !fail {
		results = append(results, common.Passf("Container %q has no sensitive mounts", container.Name))
	}
	return results
}

func sharedNetwork(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.NetworkMode.IsHost() {
		return common.FailListf("Container %q has network mode set to 'host'", container.Name)
	}
	return common.PassListf("Container %q has network mode set to %q", container.Name, container.HostConfig.NetworkMode)
}

func ulimit(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if len(container.HostConfig.Ulimits) > 0 {
		var ulimits []string
		for _, u := range container.HostConfig.Ulimits {
			ulimits = append(ulimits, fmt.Sprintf("(name=%q; hard=%q; soft=%q)", u.Name, u.Hard, u.Soft))
		}
		return common.FailListf("Container %q overrides ulimits %s", container.Name, msgfmt.FormatStrings(ulimits...))
	}
	return common.PassListf("Container %q does not override ulimits", container.Name)
}

func userNamespace(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.UsernsMode.IsHost() {
		return common.FailListf("Container %q has user namespace mode set to 'host'", container.Name)
	}
	return common.PassListf("Container %q has user namespace mode set to %q", container.Name, container.HostConfig.UsernsMode)
}

func utsNamespace(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	if container.HostConfig.UTSMode.IsHost() {
		return common.FailListf("Container %q has UTS namespace mode set to 'host'", container.Name)
	}
	return common.PassListf("Container %q has UTS namespace mode set to %q", container.Name, container.HostConfig.UTSMode)
}

func isRootUser(user string) bool {
	return user == "" || user == "root" || user == "0"
}

func usersInContainer(container types.ContainerJSON) []*storage.ComplianceResultValue_Evidence {
	user := container.Config.User

	if isRootUser(user) {
		return common.FailListf("Container %q is running as the root user", container.Name)
	}

	user, _ = stringutils.Split2(user, ":")
	if isRootUser(user) {
		return common.FailListf("Container %q is running as the root user", container.Name)
	}
	return common.PassListf("Container %q is running as the user %q", container.Name, user)
}
