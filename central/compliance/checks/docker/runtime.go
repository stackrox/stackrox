package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/set"
)

func init() {
	framework.MustRegisterChecks(
		runningContainerCheck("CIS_Docker_v1_1_0:5_1", appArmor, "has an AppArmor profile configured"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_2", selinux, "has SELinux configured"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_3", capabilities, "has extra capabilities enabled"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_4", privileged, "is not running in privileged mode"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_5", sensitiveHostMounts, "does not mount any sensitive host directories"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:5_6", "Check containers to ensure SSH is not running within them"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_7", privilegedPorts, "does not bind to a privileged host port"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_8", necessaryPorts, "does not bind to any host ports"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_9", sharedNetwork, "does not use the 'host' network mode"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_10", memoryLimit, "has memory limits configured"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_11", cpuShares, "has CPU shares configured"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_12", readonlyFS, "uses a read-only root filesystem"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_13", specificHostInterface, "does not bind to all host interface addresses (0.0.0.0)"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_14", restartPolicy, "has an on-failure restart policy with a maximum of 5 retries"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_15", pidNamespace, "is not using the host PID namespace"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_16", ipcNamespace, "is not using the host IPC namespace"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_17", hostDevices, "has not mounted any host devices"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_18", ulimit, "does not override ulimits"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_19", mountPropagation, "does not have any mounts that use shared propagation"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_20", utsNamespace, "does not use the host UTS namespace"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_21", seccomp, "does not have seccomp set to unconfined"),

		// 5.22 and 5.23 are in file.go

		runningContainerCheck("CIS_Docker_v1_1_0:5_24", cgroup, "does not use a non-standard cgroup parent"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_25", acquiringPrivileges, "sets no-new-privileges in its security options"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_26", healthcheck, "has a correctly configured health check"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:5_27", "Pulling images is invasive and not always possible depending on credential management"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_28", pidCgroup, "has a PID limit set"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_29", bridgeNetwork, "is not running on the bridge network"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_30", userNamespace, "is not using the host user namespace"),
		runningContainerCheck("CIS_Docker_v1_1_0:5_31", noDockerSocket, "does not mount the docker socket"),

		// One off
		runningContainerCheck("CIS_Docker_v1_1_0:4_1", usersInContainer, "is not running as the root user"),
	)
}

func runningContainerCheck(name string, f func(ctx framework.ComplianceContext, container docker.ContainerJSON), desc string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that every running container on each node %s", desc),
	}
	return framework.NewCheckFromFunc(md, containerCheckWrapper(f, true))
}

func containerCheckWrapper(f func(ctx framework.ComplianceContext, container docker.ContainerJSON), runningOnly bool) framework.CheckFunc {
	return common.PerNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *docker.Data) {
		for _, c := range data.Containers {
			if runningOnly && (c.State == nil || !c.State.Running) {
				continue
			}
			if c.HostConfig == nil {
				continue
			}
			f(ctx, c)
		}
	})
}

func acquiringPrivileges(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var pass bool
	for _, o := range container.HostConfig.SecurityOpt {
		if strings.Contains(o, "no-new-privileges") {
			pass = true
			framework.Passf(ctx, "Container %q sets no-new-privileges", container.Name)
		}
	}
	if !pass {
		framework.Failf(ctx, "Container %q does not set no-new-privileges in security opts", container.Name)
	}
}

func appArmor(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.AppArmorProfile == "unconfined" {
		framework.Failf(ctx, "Container %q has app armor configured as unconfined", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has app armor profile configured as %q", container.Name, container.AppArmorProfile)
	}
}

func specificHostInterface(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.NetworkSettings == nil {
		framework.Pass(ctx, "Container %q has no values set for network settings")
		return
	}

	var failed bool
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			if strings.Contains(binding.HostIP, "0.0.0.0") {
				failed = true
				framework.Failf(ctx, "Container %q binds port %d to '0.0.0.0 %s'", container.Name, containerPort.Int(), binding.HostPort)
			}
		}
	}
	if !failed {
		framework.Passf(ctx, "Container %q binds no ports to all interfaces (0.0.0.0)", container.Name)
	}
}

func bridgeNetwork(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.NetworkSettings == nil {
		framework.Passf(ctx, "Container %q has no network settings", container.Name)
		return
	}
	if _, ok := container.NetworkSettings.Networks["bridge"]; ok {
		framework.Failf(ctx, "Container %q is running on the bridge network", container.Name)
	} else {
		framework.Passf(ctx, "Container %q is not running on the bridge network", container.Name)
	}
}

func capabilities(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if len(container.HostConfig.CapAdd) > 0 {
		framework.Notef(ctx, "Container %q adds capabilities: %s", container.Name, strings.Join(container.HostConfig.CapAdd, ", "))
	} else {
		framework.Passf(ctx, "Container %q does not add any capabilities", container.Name)
	}
}

func cgroup(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.CgroupParent != "docker" && container.HostConfig.CgroupParent != "" {
		framework.Failf(ctx, "Container %q has the cgroup parent set to %s", container.Name, container.HostConfig.CgroupParent)
	} else {
		framework.Passf(ctx, "Container %q has the cgroup parent set to %q", container.Name, container.HostConfig.CgroupParent)
	}
}

func cpuShares(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.CPUShares == 0 {
		framework.Failf(ctx, "Container %q does not have CPU shares set", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has CPU shares set to %d", container.Name, container.HostConfig.CPUShares)
	}
}

func healthcheck(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.State.Health == nil {
		framework.Failf(ctx, "Container %q does not have health configured", container.Name)
	} else if container.State.Health.Status == "" {
		framework.Failf(ctx, "Container %q is currently reporting empty health", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has health configured with status %q", container.Name, container.State.Health.Status)
	}
}

func hostDevices(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if len(container.HostConfig.Devices) > 0 {
		devices := make([]string, 0, len(container.HostConfig.Devices))
		for _, device := range container.HostConfig.Devices {
			devices = append(devices, fmt.Sprintf("%v:%v", device.PathOnHost, device.PathInContainer))
		}
		framework.Failf(ctx, "Container %q has host devices [ %s ] exposed to it", container.Name, strings.Join(devices, " | "))
	} else {
		framework.Passf(ctx, "Container %q has not mounted any host devices", container.Name)
	}
}

func ipcNamespace(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.IpcMode.IsHost() {
		framework.Failf(ctx, "Container %q has IPC mode set to 'host'", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has IPC mode set to %q", container.Name, container.HostConfig.IpcMode)
	}
}

func memoryLimit(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.Memory == 0 {
		framework.Failf(ctx, "Container %q does not have a memory limit", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has a memory limit set to %d", container.Name, container.HostConfig.Memory)
	}
}

func mountPropagation(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var failed bool
	for _, containerMount := range container.Mounts {
		if containerMount.Propagation == mount.PropagationShared {
			failed = true
			framework.Failf(ctx, "Container %q has mount %q which uses shared propagation", container.Name, containerMount.Name)
		}
	}
	if !failed {
		framework.Passf(ctx, "Container %q has no mounts that use shared propagation", container.Name)
	}
}

func necessaryPorts(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.NetworkSettings == nil {
		framework.Passf(ctx, "Container %q does not have any network settings", container.Name)
		return
	}
	var failed bool
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			failed = true
			framework.Notef(ctx, "Container %q binds container port '%d' to host port %q", container.Name, containerPort.Int(), binding.HostPort)
		}
	}
	if !failed {
		framework.Passf(ctx, "Container %q does not bind any container ports to host ports", container.Name)
	}
}

// Docker socket
func noDockerSocket(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var failed bool
	for _, containerMount := range container.Mounts {
		if strings.Contains(containerMount.Source, "docker.sock") {
			failed = true
			framework.Failf(ctx, "Container %q has mounted docker.sock", container.Name)
		}
	}
	if !failed {
		framework.Passf(ctx, "Container %q has not mounted docker.sock", container.Name)
	}
}

func pidNamespace(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.PidMode.IsHost() {
		framework.Failf(ctx, "Container %q has PID mode set to 'host'", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has PID mode set to %q", container.Name, container.HostConfig.PidMode)
	}
}

func pidCgroup(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.PidsLimit <= 0 {
		framework.Failf(ctx, "Container %q does not have PIDs limit set", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has PIDs limit set to %d", container.Name, container.HostConfig.PidsLimit)
	}
}

func privileged(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.Privileged {
		framework.Failf(ctx, "Container %q is running as privileged", container.Name)
	} else {
		framework.Passf(ctx, "Container %q is not running as privileged", container.Name)
	}
}

func privilegedPorts(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var failed bool
	for containerPort, hostBinding := range container.NetworkSettings.Ports {
		for _, binding := range hostBinding {
			portNum, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				failed = true
				framework.Failf(ctx, "Could not parse host port for container %q", container.Name)
			} else if portNum < 1024 {
				failed = true
				framework.Failf(ctx, "Container %q binds port '%d' to privileged host port '%d'", container.Name, containerPort.Int(), portNum)
			}
		}
	}
	if !failed {
		framework.Passf(ctx, "Container %q does not bind any ports with numbers < 1024", container.Name)
	}
}

func readonlyFS(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if !container.HostConfig.ReadonlyRootfs {
		framework.Failf(ctx, "Container %q does not have a readonly rootFS", container.Name)
	} else {
		framework.Passf(ctx, "Container %q had a readonly rootFS", container.Name)
	}
}

func restartPolicy(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.RestartPolicy.Name != "on-failure" || container.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
		framework.Failf(ctx, "Container %q has a restart policy %q with max retries '%d'", container.Name, container.HostConfig.RestartPolicy.Name, container.HostConfig.RestartPolicy.MaximumRetryCount)
	} else {
		framework.Passf(ctx, "Container %q has the 'on-failure' restart policy with 5 maximum retries", container.Name)
	}
}

func seccomp(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var fail bool
	for _, opt := range container.HostConfig.SecurityOpt {
		if strings.EqualFold(opt, "seccomp:unconfined") {
			fail = true
			framework.Failf(ctx, "Container %q has seccomp set to unconfined", container.Name)
		}
	}
	if !fail {
		framework.Passf(ctx, "Container %q does not have seccomp set to unconfined", container.Name)
	}
}

func selinux(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var pass bool
	for _, opt := range container.HostConfig.SecurityOpt {
		if strings.Contains(opt, "selinux") {
			pass = true
			framework.Passf(ctx, "Container %q has selinux configured as %q", container.Name, opt)
		}
	}
	if !pass {
		framework.Failf(ctx, "Container %q does not have selinux configured", container.Name)
	}
}

var sensitiveMounts = set.NewStringSet(
	"/",
	"/boot",
	"/dev",
	"/etc",
	"/lib",
	"/proc",
	"/sys",
	"/usr",
)

func sensitiveHostMounts(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	var fail bool
	for _, mount := range container.Mounts {
		if sensitiveMounts.Contains(mount.Source) {
			fail = true
			framework.Failf(ctx, "Container %q has sensitive mount %q with mode %q", container.Name, mount.Source, mount.Mode)
		}
	}
	if !fail {
		framework.Passf(ctx, "Container %q has no sensitive mounts", container.Name)
	}
}

func sharedNetwork(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.NetworkMode.IsHost() {
		framework.Failf(ctx, "Container %q has network mode set to 'host'", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has network mode set to %q", container.Name, container.HostConfig.NetworkMode)
	}
}

func ulimit(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if len(container.HostConfig.Ulimits) > 0 {
		var ulimits []string
		for _, u := range container.HostConfig.Ulimits {
			ulimits = append(ulimits, fmt.Sprintf("(name=%q; hard=%q; soft=%q)", u.Name, u.Hard, u.Soft))
		}
		framework.Failf(ctx, "Container %q overrides ulimits %s", container.Name, msgfmt.FormatStrings(ulimits...))
	} else {
		framework.Passf(ctx, "Container %q does not override ulimits", container.Name)
	}
}

func userNamespace(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.UsernsMode.IsHost() {
		framework.Failf(ctx, "Container %q has user namespace mode set to 'host'", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has user namespace mode set to %q", container.Name, container.HostConfig.UsernsMode)
	}
}

func utsNamespace(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.HostConfig.UTSMode.IsHost() {
		framework.Failf(ctx, "Container %q has UTS namespace mode set to 'host'", container.Name)
	} else {
		framework.Passf(ctx, "Container %q has UTS namespace mode set to %q", container.Name, container.HostConfig.UTSMode)
	}
}

func usersInContainer(ctx framework.ComplianceContext, container docker.ContainerJSON) {
	if container.Config != nil && (container.Config.User == "" || container.Config.User == "root") {
		framework.Failf(ctx, "Container %q is running as the root user", container.Name)
	} else {
		framework.Passf(ctx, "Container %q is running as the user %q", container.Name, container.Config.User)
	}
}
