package hostconfiguration

import "bitbucket.org/stack-rox/apollo/docker-bench/common"

// NewDockerDaemonAudit implements CIS-1.5
func NewDockerDaemonAudit() common.Benchmark {
	return newPathAudit("CIS 1.5", "Ensure auditing is configured for the docker daemon", "/usr/bin/docker")
}

// NewVarLibDockerAudit implements CIS-1.6
func NewVarLibDockerAudit() common.Benchmark {
	return newPathAudit("CIS 1.6", "Ensure auditing is configured for Docker files and directories - /var/lib/docker", "/var/lib/docker")
}

// NewEtcDockerAudit implements CIS-1.7
func NewEtcDockerAudit() common.Benchmark {
	return newPathAudit("CIS 1.7", "Ensure auditing is configured for Docker files and directories - /etc/docker", "/etc/docker")
}

// NewDockerServiceAudit implements CIS-1.8
func NewDockerServiceAudit() common.Benchmark {
	return newSystemdAudit("CIS 1.8", "Ensure auditing is configured for Docker files and directories - docker.service", "docker.service")
}

// NewDockerSocketAudit implements CIS-1.9
func NewDockerSocketAudit() common.Benchmark {
	return newSystemdAudit("CIS 1.9", "Ensure auditing is configured for Docker files and directories - docker.socket", "docker.socket")
}

// NewEtcDefaultDockerAudit implements CIS-1.10
func NewEtcDefaultDockerAudit() common.Benchmark {
	return newPathAudit("CIS 1.10", "Ensure auditing is configured for Docker files and directories - /etc/default/docker", "/etc/default/docker")
}

// NewEtcDockerDaemonJSONAudit implements CIS-1.11
func NewEtcDockerDaemonJSONAudit() common.Benchmark {
	return newPathAudit("CIS 1.11", "Ensure auditing is configured for Docker files and directories - /etc/docker/daemon.json", "/etc/docker/daemon.json")
}

// NewUsrBinContainerdAudit implements CIS-1.12
func NewUsrBinContainerdAudit() common.Benchmark {
	return newPathAudit("CIS 1.12", "Ensure auditing is configured for Docker files and directories - /usr/bin/docker-containerd", "/usr/bin/docker-containerd")
}

// NewUsrBinRundAudit implements CIS-1.13
func NewUsrBinRundAudit() common.Benchmark {
	return newPathAudit("CIS 1.13", "Ensure auditing is configured for Docker files and directories - /usr/bin/docker-runc", "/usr/bin/docker-runc")
}
