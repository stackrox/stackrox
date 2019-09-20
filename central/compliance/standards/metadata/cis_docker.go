package metadata

var cisDocker = Standard{
	ID:   "CIS_Docker_v1_2_0",
	Name: "CIS Docker v1.2.0",
	Categories: []Category{
		{
			ID:          "1.1",
			Name:        "1.1",
			Description: "Host Configuration - General Configuration",
			Controls: []Control{
				{
					ID:          "1_1_1",
					Name:        "1.1.1",
					Description: "Ensure the container host has been Hardened",
				},
				{
					ID:          "1_1_2",
					Name:        "1.1.2",
					Description: "Ensure that the version of Docker is up to date",
				},
			},
		},
		{
			ID:          "1.2",
			Name:        "1.2",
			Description: "Host Configuration - Linux Hosts Specific Configuration",
			Controls: []Control{
				{
					ID:          "1_2_1",
					Name:        "1.2.1",
					Description: "Ensure a separate partition for containers has been created",
				},
				{
					ID:          "1_2_2",
					Name:        "1.2.2",
					Description: "Ensure only trusted users are allowed to control Docker daemon",
				},
				{
					ID:          "1_2_3",
					Name:        "1.2.3",
					Description: "Ensure auditing is configured for the docker daemon",
				},
				{
					ID:          "1_2_4",
					Name:        "1.2.4",
					Description: "Ensure auditing is configured for Docker files and directories - /var/lib/docker",
				},
				{
					ID:          "1_2_5",
					Name:        "1.2.5",
					Description: "Ensure auditing is configured for Docker files and directories - /etc/docker",
				},
				{
					ID:          "1_2_6",
					Name:        "1.2.6",
					Description: "Ensure auditing is configured for Docker files and directories - docker.service",
				},
				{
					ID:          "1_2_7",
					Name:        "1.2.7",
					Description: "Ensure auditing is configured for Docker files and directories - docker.socket",
				},
				{
					ID:          "1_2_8",
					Name:        "1.2.8",
					Description: "Ensure auditing is configured for Docker files and directories - /etc/default/docker",
				},
				{
					ID:          "1_2_9",
					Name:        "1.2.9",
					Description: "Ensure auditing is configured for Docker files and directories - /etc/sysconfig/docker",
				},
				{
					ID:          "1_2_10",
					Name:        "1.2.10",
					Description: "Ensure auditing is configured for Docker files and directories - /etc/docker/daemon.json",
				},
				{
					ID:          "1_2_11",
					Name:        "1.2.11",
					Description: "Ensure auditing is configured for Docker files and directories - /usr/bin/docker-containerd",
				},
				{
					ID:          "1_2_12",
					Name:        "1.2.12",
					Description: "Ensure auditing is configured for Docker files and directories - /usr/sbin/runc",
				},
			},
		},
		{
			ID:          "2",
			Name:        "2",
			Description: "Docker Daemon Configuration",
			Controls: []Control{
				{
					ID:          "2_1",
					Name:        "2.1",
					Description: "Ensure network traffic is restricted between containers on the default bridge",
				},
				{
					ID:          "2_2",
					Name:        "2.2",
					Description: "Ensure the logging level is set to 'info'",
				},
				{
					ID:          "2_3",
					Name:        "2.3",
					Description: "Ensure Docker is allowed to make changes to iptables",
				},
				{
					ID:          "2_4",
					Name:        "2.4",
					Description: "Ensure insecure registries are not used",
				},
				{
					ID:          "2_5",
					Name:        "2.5",
					Description: "Ensure aufs storage driver is not used",
				},
				{
					ID:          "2_6",
					Name:        "2.6",
					Description: "Ensure TLS authentication for Docker daemon is configured",
				},
				{
					ID:          "2_7",
					Name:        "2.7",
					Description: "Ensure the default ulimit is configured appropriately",
				},
				{
					ID:          "2_8",
					Name:        "2.8",
					Description: "Enable user namespace support",
				},
				{
					ID:          "2_9",
					Name:        "2.9",
					Description: "Ensure the default cgroup usage has been confirmed",
				},
				{
					ID:          "2_10",
					Name:        "2.10",
					Description: "Ensure base device size is not changed until needed",
				},
				{
					ID:          "2_11",
					Name:        "2.11",
					Description: "Ensure that authorization for Docker client commands is enabled",
				},
				{
					ID:          "2_12",
					Name:        "2.12",
					Description: "Ensure centralized and remote logging is configured",
				},
				{
					ID:          "2_13",
					Name:        "2.13",
					Description: "Ensure live restore is Enabled",
				},
				{
					ID:          "2_14",
					Name:        "2.14",
					Description: "Ensure Userland Proxy is Disabled",
				},
				{
					ID:          "2_15",
					Name:        "2.15",
					Description: "Ensure that a daemon-wide custom seccomp profile is applied, if appropriate",
				},
				{
					ID:          "2_16",
					Name:        "2.16",
					Description: "Ensure that experimental features are not implemented in production",
				},
				{
					ID:          "2_17",
					Name:        "2.17",
					Description: "Ensure containers are restricted from acquiring new privileges",
				},
			},
		},
		{
			ID:          "3",
			Name:        "3",
			Description: "Docker Daemon Configuration Files",
			Controls: []Control{
				{
					ID:          "3_1",
					Name:        "3.1",
					Description: "Ensure that the docker.service file ownership is set to root:root",
				},
				{
					ID:          "3_2",
					Name:        "3.2",
					Description: "Ensure that docker.service file permissions are appropriately set",
				},
				{
					ID:          "3_3",
					Name:        "3.3",
					Description: "Ensure that docker.socket file ownership is set to root:root",
				},
				{
					ID:          "3_4",
					Name:        "3.4",
					Description: "Ensure that docker.socket file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "3_5",
					Name:        "3.5",
					Description: "Ensure that the /etc/docker directory ownership is set to root:root",
				},
				{
					ID:          "3_6",
					Name:        "3.6",
					Description: "Ensure that /etc/docker directory permissions are set to 755 or more restrictive",
				},
				{
					ID:          "3_7",
					Name:        "3.7",
					Description: "Ensure that registry certificate file ownership is set to root:root",
				},
				{
					ID:          "3_8",
					Name:        "3.8",
					Description: "Ensure that registry certificate file permissions are set to 444 or more restrictive",
				},
				{
					ID:          "3_9",
					Name:        "3.9",
					Description: "Ensure that TLS CA certificate file ownership is set to root:root",
				},
				{
					ID:          "3_10",
					Name:        "3.10",
					Description: "Ensure that TLS CA certificate file permissions are set to 444 or more restrictive",
				},
				{
					ID:          "3_11",
					Name:        "3.11",
					Description: "Ensure that Docker server certificate file ownership is set to root:root",
				},
				{
					ID:          "3_12",
					Name:        "3.12",
					Description: "Ensure that the Docker server certificate file permissions are set to 444 or more restrictive",
				},
				{
					ID:          "3_13",
					Name:        "3.13",
					Description: "Ensure that the Docker server certificate key file ownership is set to root:root",
				},
				{
					ID:          "3_14",
					Name:        "3.14",
					Description: "Ensure that the Docker server certificate key file permissions are set to 400",
				},
				{
					ID:          "3_15",
					Name:        "3.15",
					Description: "Ensure that the Docker socket file ownership is set to root:docker",
				},
				{
					ID:          "3_16",
					Name:        "3.16",
					Description: "Ensure that Docker socket file permissions are set to 660 or more restrictive",
				},
				{
					ID:          "3_17",
					Name:        "3.17",
					Description: "Ensure that the daemon.json file ownership is set to root:root",
				},
				{
					ID:          "3_18",
					Name:        "3.18",
					Description: "Ensure that daemon.json file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "3_19",
					Name:        "3.19",
					Description: "Ensure that the /etc/default/docker file ownership is set to root:root",
				},
				{
					ID:          "3_20",
					Name:        "3.20",
					Description: "Ensure that the /etc/sysconfig/docker file ownership is set to root:root",
				},
				{
					ID:          "3_21",
					Name:        "3.21",
					Description: "Ensure that the /etc/sysconfig/docker file permissions are set to 644 or more restrictively",
				},
				{
					ID:          "3_22",
					Name:        "3.22",
					Description: "Ensure that the /etc/default/docker file permissions are set to 644 or more restrictive",
				},
			},
		},
		{
			ID:          "4",
			Name:        "4",
			Description: "Container Images and Build File",
			Controls: []Control{
				{
					ID:          "4_1",
					Name:        "4.1",
					Description: "Ensure that a user for the container has been created",
				},
				{
					ID:          "4_2",
					Name:        "4.2",
					Description: "Ensure that containers use only trusted base images",
				},
				{
					ID:          "4_3",
					Name:        "4.3",
					Description: "Ensure that unnecessary packages are not installed in the container",
				},
				{
					ID:          "4_4",
					Name:        "4.4",
					Description: "Ensure images are scanned and rebuilt to include security patches",
				},
				{
					ID:          "4_5",
					Name:        "4.5",
					Description: "Ensure Content trust for Docker is Enabled",
				},
				{
					ID:          "4_6",
					Name:        "4.6",
					Description: "Ensure that HEALTHCHECK instructions have been added to the container image",
				},
				{
					ID:          "4_7",
					Name:        "4.7",
					Description: "Ensure update instructions are not use alone in the Dockerfile",
				},
				{
					ID:          "4_8",
					Name:        "4.8",
					Description: "Ensure setuid and setgid permissions are removed",
				},
				{
					ID:          "4_9",
					Name:        "4.9",
					Description: "Ensure that COPY is used instead of ADD in Dockerfiles",
				},
				{
					ID:          "4_10",
					Name:        "4.10",
					Description: "Ensure secrets are not stored in Dockerfiles",
				},
				{
					ID:          "4_11",
					Name:        "4.11",
					Description: "Ensure only verified packages are installed",
				},
			},
		},
		{
			ID:          "5",
			Name:        "5",
			Description: "Container Runtime",
			Controls: []Control{
				{
					ID:          "5_1",
					Name:        "5.1",
					Description: "Ensure that, if applicable, an AppArmor Profile is Enabled",
				},
				{
					ID:          "5_2",
					Name:        "5.2",
					Description: "Ensure that, if applicable, SELinux security options are set",
				},
				{
					ID:          "5_3",
					Name:        "5.3",
					Description: "Ensure that Linux Kernel Capabilities are restricted within containers",
				},
				{
					ID:          "5_4",
					Name:        "5.4",
					Description: "Ensure that privileged containers are not used",
				},
				{
					ID:          "5_5",
					Name:        "5.5",
					Description: "Ensure sensitive host system directories are not mounted on containers",
				},
				{
					ID:          "5_6",
					Name:        "5.6",
					Description: "Ensure sshd is not run within containers",
				},
				{
					ID:          "5_7",
					Name:        "5.7",
					Description: "Ensure privileged ports are not mapped within containers",
				},
				{
					ID:          "5_8",
					Name:        "5.8",
					Description: "Ensure that only needed ports are open on the container",
				},
				{
					ID:          "5_9",
					Name:        "5.9",
					Description: "Ensure that the host's network namespace is not shared",
				},
				{
					ID:          "5_10",
					Name:        "5.10",
					Description: "Ensure that the memory usage for container is limited",
				},
				{
					ID:          "5_11",
					Name:        "5.11",
					Description: "Ensure that CPU priority is set appropriately on the container",
				},
				{
					ID:          "5_12",
					Name:        "5.12",
					Description: "Ensure that the container's root filesystem is mounted as read only",
				},
				{
					ID:          "5_13",
					Name:        "5.13",
					Description: "Ensure that incoming container traffic is bound to a specific host interface",
				},
				{
					ID:          "5_14",
					Name:        "5.14",
					Description: "Ensure that the 'on-failure' container restart policy is set to '5'",
				},
				{
					ID:          "5_15",
					Name:        "5.15",
					Description: "Ensure that the host's process namespace is not shared",
				},
				{
					ID:          "5_16",
					Name:        "5.16",
					Description: "Ensure that the host's IPC namespace is not shared",
				},
				{
					ID:          "5_17",
					Name:        "5.17",
					Description: "Ensure that host devices are not directly exposed to containers",
				},
				{
					ID:          "5_18",
					Name:        "5.18",
					Description: "Ensure that the default ulimit is overwritten at runtime, if needed",
				},
				{
					ID:          "5_19",
					Name:        "5.19",
					Description: "Ensure mount propagation mode is not set to shared",
				},
				{
					ID:          "5_20",
					Name:        "5.20",
					Description: "Ensure that the host's UTS namespace is not shared",
				},
				{
					ID:          "5_21",
					Name:        "5.21",
					Description: "Ensure the default seccomp profile is not Disabled",
				},
				{
					ID:          "5_22",
					Name:        "5.22",
					Description: "Ensure that docker exec commands are not used with privileged option",
				},
				{
					ID:          "5_23",
					Name:        "5.23",
					Description: "Ensure that docker exec commands are not used with user=root option",
				},
				{
					ID:          "5_24",
					Name:        "5.24",
					Description: "Ensure that cgroup usage is confirmed",
				},
				{
					ID:          "5_25",
					Name:        "5.25",
					Description: "Ensure that the container is restricted from acquiring additional privileges",
				},
				{
					ID:          "5_26",
					Name:        "5.26",
					Description: "Ensure that container health is checked at runtime",
				},
				{
					ID:          "5_27",
					Name:        "5.27",
					Description: "Ensure that Docker commands always make use of the latest version of their image",
				},
				{
					ID:          "5_28",
					Name:        "5.28",
					Description: "Ensure that the PIDs cgroup limit is used",
				},
				{
					ID:          "5_29",
					Name:        "5.29",
					Description: "Ensure that Docker's default bridge \"docker0\" is not used",
				},
				{
					ID:          "5_30",
					Name:        "5.30",
					Description: "Ensure that the host's user namespaces are not shared",
				},
				{
					ID:          "5_31",
					Name:        "5.31",
					Description: "Ensure that the Docker socket is not mounted inside any containers",
				},
			},
		},
		{
			ID:          "6",
			Name:        "6",
			Description: "Docker Security Operations",
			Controls: []Control{
				{
					ID:          "6_1",
					Name:        "6.1",
					Description: "Ensure that image sprawl is avoided",
				},
				{
					ID:          "6_2",
					Name:        "6.2",
					Description: "Ensure that container sprawl is avoided",
				},
			},
		},
		{
			ID:          "7",
			Name:        "7",
			Description: "Docker Swarm Configuration",
			Controls: []Control{
				{
					ID:          "7_1",
					Name:        "7.1",
					Description: "Ensure swarm mode is not Enabled, if not needed",
				},
				{
					ID:          "7_2",
					Name:        "7.2",
					Description: "Ensure that the minimum number of manager nodes have been created in a swarm",
				},
				{
					ID:          "7_3",
					Name:        "7.3",
					Description: "Ensure that swarm services are bound to a specific host interface",
				},
				{
					ID:          "7_4",
					Name:        "7.4",
					Description: "Ensure that all Docker swarm overlay networks are encrypted",
				},
				{
					ID:          "7_5",
					Name:        "7.5",
					Description: "Ensure that Docker's secret management commands are used for managing secrets in a swarm cluster",
				},
				{
					ID:          "7_6",
					Name:        "7.6",
					Description: "Ensure that swarm manager is run in auto-lock mode",
				},
				{
					ID:          "7_7",
					Name:        "7.7",
					Description: "Ensure that the swarm manager auto-lock key is rotated periodically",
				},
				{
					ID:          "7_8",
					Name:        "7.8",
					Description: "Ensure that node certificates are rotated as appropriate",
				},
				{
					ID:          "7_9",
					Name:        "7.9",
					Description: "Ensure that CA certificates are rotated as appropriate",
				},
				{
					ID:          "7_10",
					Name:        "7.10",
					Description: "Ensure that management plane traffic has been separated from data plane traffic",
				},
			},
		},
		{
			ID:          "8_1",
			Name:        "8.1",
			Description: "Docker Enterprise Configuration - Universal Control Plane Configuration",
			Controls: []Control{
				{
					ID:          "8_1_1",
					Name:        "8.1.1",
					Description: "Configure the LDAP authentication service",
				},
				{
					ID:          "8_1_2",
					Name:        "8.1.2",
					Description: "Use external certificates",
				},
				{
					ID:          "8_1_3",
					Name:        "8.1.3",
					Description: "Enforce the use of client certificate bundles for unprivileged users",
				},
				{
					ID:          "8_1_4",
					Name:        "8.1.4",
					Description: "Configure applicable cluster role-based access control policies",
				},
				{
					ID:          "8_1_5",
					Name:        "8.1.5",
					Description: "Enable signed image enforcement",
				},
				{
					ID:          "8_1_6",
					Name:        "8.1.6",
					Description: "Set the Per-User Session Limit to a value of '3' or lower",
				},
				{
					ID:          "8_1_7",
					Name:        "8.1.7",
					Description: "Set the \"Lifetime Minutes\" and \"Renewal Threshold Minutes\" values to '15' or lower and '0' respectively",
				},
			},
		},
		{
			ID:          "8_2",
			Name:        "8.2",
			Description: "Docker Enterprise Configuration - Docker Trusted Registry Configuration",
			Controls: []Control{
				{
					ID:          "8_2_1",
					Name:        "8.2.1",
					Description: "Enable image vulnerability scanning",
				},
			},
		},
	},
}

func init() {
	AllStandards = append(AllStandards, cisDocker)
}
