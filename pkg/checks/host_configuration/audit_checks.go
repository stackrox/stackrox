package hostconfiguration

import "github.com/stackrox/rox/pkg/checks/utils"

// NewDockerDaemonAudit implements CIS-1.5
func NewDockerDaemonAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.5", "Ensure auditing is configured for the docker daemon", "/usr/bin/docker")
}

// NewVarLibDockerAudit implements CIS-1.6
func NewVarLibDockerAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.6", "Ensure auditing is configured for Docker files and directories - /var/lib/docker", "/var/lib/docker")
}

// NewEtcDockerAudit implements CIS-1.7
func NewEtcDockerAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.7", "Ensure auditing is configured for Docker files and directories - /etc/docker", "/etc/docker")
}

// NewDockerServiceAudit implements CIS-1.8
func NewDockerServiceAudit() utils.Check {
	return newSystemdAudit("CIS Docker v1.1.0 - 1.8", "Ensure auditing is configured for Docker files and directories - docker.service", "docker.service")
}

// NewDockerSocketAudit implements CIS-1.9
func NewDockerSocketAudit() utils.Check {
	return newSystemdAudit("CIS Docker v1.1.0 - 1.9", "Ensure auditing is configured for Docker files and directories - docker.socket", "docker.socket")
}

// NewEtcDefaultDockerAudit implements CIS-1.10
func NewEtcDefaultDockerAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.10", "Ensure auditing is configured for Docker files and directories - /etc/default/docker", "/etc/default/docker")
}

// NewEtcDockerDaemonJSONAudit implements CIS-1.11
func NewEtcDockerDaemonJSONAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.11", "Ensure auditing is configured for Docker files and directories - /etc/docker/daemon.json", "/etc/docker/daemon.json")
}

// NewUsrBinContainerdAudit implements CIS-1.12
func NewUsrBinContainerdAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.12", "Ensure auditing is configured for Docker files and directories - /usr/bin/docker-containerd", "/usr/bin/docker-containerd")
}

// NewUsrBinRundAudit implements CIS-1.13
func NewUsrBinRundAudit() utils.Check {
	return newPathAudit("CIS Docker v1.1.0 - 1.13", "Ensure auditing is configured for Docker files and directories - /usr/bin/docker-runc", "/usr/bin/docker-runc")
}
