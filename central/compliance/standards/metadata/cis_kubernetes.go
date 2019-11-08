package metadata

var cisKubernetes = Standard{
	ID:   "CIS_Kubernetes_v1_5",
	Name: "CIS Kubernetes v1.5",
	Categories: []Category{
		{
			ID:          "1_1",
			Name:        "1.1",
			Description: "Master Node Security Configuration - Configuration Files",
			Controls: []Control{
				{
					ID:          "1_1_1",
					Name:        "1.1.1",
					Description: "Ensure that the API server pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_2",
					Name:        "1.1.2",
					Description: "Ensure that the API server pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_1_3",
					Name:        "1.1.3",
					Description: "Ensure that the controller manager pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_4",
					Name:        "1.1.4",
					Description: "Ensure that the controller manager pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_1_5",
					Name:        "1.1.5",
					Description: "Ensure that the scheduler pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_6",
					Name:        "1.1.6",
					Description: "Ensure that the scheduler pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_1_7",
					Name:        "1.1.7",
					Description: "Ensure that the etcd pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_8",
					Name:        "1.1.8",
					Description: "Ensure that the etcd pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_1_9",
					Name:        "1.1.9",
					Description: "Ensure that the Container Network Interface file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_10",
					Name:        "1.1.10",
					Description: "Ensure that the Container Network Interface file ownership is set to root:root",
				},
				{
					ID:          "1_1_11",
					Name:        "1.1.11",
					Description: "Ensure that the etcd data directory permissions are set to 700 or more restrictive",
				},
				{
					ID:          "1_1_12",
					Name:        "1.1.12",
					Description: "Ensure that the etcd data directory ownership is set to etcd:etcd",
				},
				{
					ID:          "1_1_13",
					Name:        "1.1.13",
					Description: "Ensure that the admin.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_14",
					Name:        "1.1.14",
					Description: "Ensure that the admin.conf file ownership is set to root:root",
				},
				{
					ID:          "1_1_15",
					Name:        "1.1.15",
					Description: "Ensure that the scheduler.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_16",
					Name:        "1.1.16",
					Description: "Ensure that the scheduler.conf file ownership is set to root:root",
				},
				{
					ID:          "1_1_17",
					Name:        "1.1.17",
					Description: "Ensure that the controller-manager.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_18",
					Name:        "1.1.18",
					Description: "Ensure that the controller-manager.conf file ownership is set to root:root",
				},
				{
					ID:          "1_1_19",
					Name:        "1.1.19",
					Description: "Ensure that the Kubernetes PKI directory and file ownership is set to root:root",
				},
				{
					ID:          "1_1_20",
					Name:        "1.1.20",
					Description: "Ensure that the Kubernetes PKI certificate file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_1_21",
					Name:        "1.1.21",
					Description: "Ensure that the Kubernetes PKI key file permissions are set to 600",
				},
			},
		},
		{
			ID:          "1_2",
			Name:        "1.2",
			Description: "Master Node Security Configuration - API Server",
			Controls: []Control{
				{
					ID:          "1_2_1",
					Name:        "1.2.1",
					Description: "Ensure that the --anonymous-auth argument is set to false",
				},
				{
					ID:          "1_2_2",
					Name:        "1.2.2",
					Description: "Ensure that the --basic-auth-file argument is not set",
				},
				{
					ID:          "1_2_3",
					Name:        "1.2.3",
					Description: "Ensure that the --token-auth-file parameter is not set",
				},
				{
					ID:          "1_2_4",
					Name:        "1.2.4",
					Description: "Ensure that the --kubelet-https argument is set to true",
				},
				{
					ID:          "1_2_5",
					Name:        "1.2.5",
					Description: "Ensure that the --kubelet-client-certificate and --kubelet-client-key arguments are set as appropriate",
				},
				{
					ID:          "1_2_6",
					Name:        "1.2.6",
					Description: "Ensure that the --kubelet-certificate-authority argument is set as appropriate",
				},
				{
					ID:          "1_2_7",
					Name:        "1.2.7",
					Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",
				},
				{
					ID:          "1_2_8",
					Name:        "1.2.8",
					Description: "Ensure that the --authorization-mode argument includes Node",
				},
				{
					ID:          "1_2_9",
					Name:        "1.2.9",
					Description: "Ensure that the --authorization-mode argument includes RBAC",
				},
				{
					ID:          "1_2_10",
					Name:        "1.2.10",
					Description: "Ensure that the admission control plugin EventRateLimit is set",
				},
				{
					ID:          "1_2_11",
					Name:        "1.2.11",
					Description: "Ensure that the admission control plugin AlwaysAdmit is not set",
				},
				{
					ID:          "1_2_12",
					Name:        "1.2.12",
					Description: "Ensure that the admission control plugin AlwaysPullImages is set",
				},
				{
					ID:          "1_2_13",
					Name:        "1.2.13",
					Description: "Ensure that the admission control plugin SecurityContextDeny is set if PodSecurityPolicy is not used",
				},
				{
					ID:          "1_2_14",
					Name:        "1.2.14",
					Description: "Ensure that the admission control plugin ServiceAccount is set",
				},
				{
					ID:          "1_2_15",
					Name:        "1.2.15",
					Description: "Ensure that the admission control plugin NamespaceLifecycle is set",
				},
				{
					ID:          "1_2_16",
					Name:        "1.2.16",
					Description: "Ensure that the admission control plugin PodSecurityPolicy is set",
				},
				{
					ID:          "1_2_17",
					Name:        "1.2.17",
					Description: "Ensure that the admission control plugin NodeRestriction is set",
				},
				{
					ID:          "1_2_18",
					Name:        "1.2.18",
					Description: "Ensure that the --insecure-bind-address argument is not set",
				},
				{
					ID:          "1_2_19",
					Name:        "1.2.19",
					Description: "Ensure that the --insecure-port argument is set to 0",
				},
				{
					ID:          "1_2_20",
					Name:        "1.2.20",
					Description: "Ensure that the --secure-port argument is not set to 0",
				},
				{
					ID:          "1_2_21",
					Name:        "1.2.21",
					Description: "Ensure that the --profiling argument is set to false",
				},
				{
					ID:          "1_2_22",
					Name:        "1.2.22",
					Description: "Ensure that the --audit-log-path argument is set",
				},
				{
					ID:          "1_2_23",
					Name:        "1.2.23",
					Description: "Ensure that the --audit-log-maxage argument is set to 30 or as appropriate",
				},
				{
					ID:          "1_2_24",
					Name:        "1.2.24",
					Description: "Ensure that the --audit-log-maxbackup argument is set to 10 or as appropriate",
				},
				{
					ID:          "1_2_25",
					Name:        "1.2.25",
					Description: "Ensure that the --audit-log-maxsize argument is set to 100 or as appropriate",
				},
				{
					ID:          "1_2_26",
					Name:        "1.2.26",
					Description: "Ensure that the --request-timeout argument is set as appropriate",
				},
				{
					ID:          "1_2_27",
					Name:        "1.2.27",
					Description: "Ensure that the --service-account-lookup argument is set to true",
				},
				{
					ID:          "1_2_28",
					Name:        "1.2.28",
					Description: "Ensure that the --service-account-key-file argument is set as appropriate",
				},
				{
					ID:          "1_2_29",
					Name:        "1.2.29",
					Description: "Ensure that the --etcd-certfile and --etcd-keyfile arguments are set as appropriate",
				},
				{
					ID:          "1_2_30",
					Name:        "1.2.30",
					Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
				},
				{
					ID:          "1_2_31",
					Name:        "1.2.31",
					Description: "Ensure that the --client-ca-file argument is set as appropriate",
				},
				{
					ID:          "1_2_32",
					Name:        "1.2.32",
					Description: "Ensure that the --etcd-cafile argument is set as appropriate",
				},
				{
					ID:          "1_2_33",
					Name:        "1.2.33",
					Description: "Ensure that the --encryption-provider-config argument is set as appropriate",
				},
				{
					ID:          "1_2_34",
					Name:        "1.2.34",
					Description: "Ensure that encryption providers are appropriately configured",
				},
				{
					ID:          "1_2_35",
					Name:        "1.2.35",
					Description: "Ensure that the API Server only makes use of Strong Cryptographic Ciphers",
				},
			},
		},
		{
			ID:          "1_3",
			Name:        "1.3",
			Description: "Master Node Security Configuration - Controller Manager",
			Controls: []Control{
				{
					ID:          "1_3_1",
					Name:        "1.3.1",
					Description: "Ensure that the --terminated-pod-gc-threshold argument is set as appropriate",
				},
				{
					ID:          "1_3_2",
					Name:        "1.3.2",
					Description: "Ensure that the --profiling argument is set to false",
				},
				{
					ID:          "1_3_3",
					Name:        "1.3.3",
					Description: "Ensure that the --use-service-account-credentials argument is set to true",
				},
				{
					ID:          "1_3_4",
					Name:        "1.3.4",
					Description: "Ensure that the --service-account-private-key-file argument is set as appropriate",
				},
				{
					ID:          "1_3_5",
					Name:        "1.3.5",
					Description: "Ensure that the --root-ca-file argument is set as appropriate",
				},
				{
					ID:          "1_3_6",
					Name:        "1.3.6",
					Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",
				},
				{
					ID:          "1_3_7",
					Name:        "1.3.7",
					Description: "Ensure that the --bind-address argument is set to 127.0.0.1",
				},
			},
		},
		{
			ID:          "1_4",
			Name:        "1.4",
			Description: "Master Node Security Configuration - Scheduler",
			Controls: []Control{
				{
					ID:          "1_4_1",
					Name:        "1.4.1",
					Description: "Ensure that the --profiling argument is set to false",
				},
				{
					ID:          "1_4_2",
					Name:        "1.4.2",
					Description: "Ensure that the --bind-address argument is set to 127.0.0.1",
				},
			},
		},
		{
			ID:          "2",
			Name:        "2",
			Description: "Master Node Security Configuration - etcd",
			Controls: []Control{
				{
					ID:          "2_1",
					Name:        "2.1",
					Description: "Ensure that the --cert-file and --key-file arguments are set as appropriate",
				},
				{
					ID:          "2_2",
					Name:        "2.2",
					Description: "Ensure that the --client-cert-auth argument is set to true",
				},
				{
					ID:          "2_3",
					Name:        "2.3",
					Description: "Ensure that the --auto-tls argument is not set to true",
				},
				{
					ID:          "2_4",
					Name:        "2.4",
					Description: "Ensure that the --peer-cert-file and --peer-key-file arguments are set as appropriate",
				},
				{
					ID:          "2_5",
					Name:        "2.5",
					Description: "Ensure that the --peer-client-cert-auth argument is set to true",
				},
				{
					ID:          "2_6",
					Name:        "2.6",
					Description: "Ensure that the --peer-auto-tls argument is not set to true",
				},
				{
					ID:          "2_7",
					Name:        "2.7",
					Description: "Ensure that a unique Certificate Authority is used for etcd",
				},
			},
		},
		{
			ID:          "3_1",
			Name:        "3.1",
			Description: "Control Plane Configuration - Authentication and Authorization",
			Controls: []Control{
				{
					ID:          "3_1_1",
					Name:        "3.1.1",
					Description: "Client certificate authentication should not be used for users",
				},
			},
		},
		{
			ID:          "3_2",
			Name:        "3.2",
			Description: "Control Plane Configuration - Logging",
			Controls: []Control{
				{
					ID:          "3_2_1",
					Name:        "3.2.1",
					Description: "Ensure that a minimal audit policy is created",
				},
				{
					ID:          "3_2_2",
					Name:        "3.2.2",
					Description: "Ensure that the audit policy covers key security concerns",
				},
			},
		},
		{
			ID:          "4_1",
			Name:        "4.1",
			Description: "Worker Node Security Configuration - Configuration Files",
			Controls: []Control{
				{
					ID:          "4_1_1",
					Name:        "4.1.1",
					Description: "Ensure that the kubelet service file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "4_1_2",
					Name:        "4.1.2",
					Description: "Ensure that the kubelet service file ownership is set to root:root",
				},
				{
					ID:          "4_1_3",
					Name:        "4.1.3",
					Description: "Ensure that the proxy kubeconfig file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "4_1_4",
					Name:        "4.1.4",
					Description: "Ensure that the proxy kubeconfig file ownership is set to root:root",
				},
				{
					ID:          "4_1_5",
					Name:        "4.1.5",
					Description: "Ensure that the kubelet.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "4_1_6",
					Name:        "4.1.6",
					Description: "Ensure that the kubelet.conf file ownership is set to root:root",
				},
				{
					ID:          "4_1_7",
					Name:        "4.1.7",
					Description: "Ensure that the certificate authorities file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "4_1_8",
					Name:        "4.1.8",
					Description: "Ensure that the client certificate authorities file ownership is set to root:root",
				},
				{
					ID:          "4_1_9",
					Name:        "4.1.9",
					Description: "Ensure that the kubelet configuration file has permissions set to 644 or more restrictive",
				},
				{
					ID:          "4_1_10",
					Name:        "4.1.10",
					Description: "Ensure that the kubelet configuration file ownership is set to root:root",
				},
			},
		},
		{
			ID:          "4_2",
			Name:        "4.2",
			Description: "Worker Node Security Configuration - Kubelet",
			Controls: []Control{
				{
					ID:          "4_2_1",
					Name:        "4.2.1",
					Description: "Ensure that the --anonymous-auth argument is set to false",
				},
				{
					ID:          "4_2_2",
					Name:        "4.2.2",
					Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",
				},
				{
					ID:          "4_2_3",
					Name:        "4.2.3",
					Description: "Ensure that the --client-ca-file argument is set as appropriate",
				},
				{
					ID:          "4_2_4",
					Name:        "4.2.4",
					Description: "Ensure that the --read-only-port argument is set to 0",
				},
				{
					ID:          "4_2_5",
					Name:        "4.2.5",
					Description: "Ensure that the --streaming-connection-idle-timeout argument is not set to 0",
				},
				{
					ID:          "4_2_6",
					Name:        "4.2.6",
					Description: "Ensure that the --protect-kernel-defaults argument is set to true",
				},
				{
					ID:          "4_2_7",
					Name:        "4.2.7",
					Description: "Ensure that the --make-iptables-util-chains argument is set to true",
				},
				{
					ID:          "4_2_8",
					Name:        "4.2.8",
					Description: "Ensure that the --hostname-override argument is not set",
				},
				{
					ID:          "4_2_9",
					Name:        "4.2.9",
					Description: "Ensure that the --event-qps argument is set to 0",
				},
				{
					ID:          "4_2_10",
					Name:        "4.2.10",
					Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
				},
				{
					ID:          "4_2_11",
					Name:        "4.2.11",
					Description: "Ensure that the --rotate-certificates argument is set not to false",
				},
				{
					ID:          "4_2_12",
					Name:        "4.2.12",
					Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",
				},
				{
					ID:          "4_2_13",
					Name:        "4.2.13",
					Description: "Ensure that the Kubelet only makes use of Strong Cryptographic Ciphers",
				},
			},
		},
		{
			ID:          "5_1",
			Name:        "5.1",
			Description: "Policies - RBAC and Service Accounts",
			Controls: []Control{
				{
					ID:          "5_1_1",
					Name:        "5.1.1",
					Description: "Ensure that the cluster-admin role is only used where required",
				},
				{
					ID:          "5_1_2",
					Name:        "5.1.2",
					Description: "Minimize access to secrets",
				},
				{
					ID:          "5_1_3",
					Name:        "5.1.3",
					Description: "Minimize wildcard use in Roles and ClusterRoles",
				},
				{
					ID:          "5_1_4",
					Name:        "5.1.4",
					Description: "Minimize access to create pods",
				},
				{
					ID:          "5_1_5",
					Name:        "5.1.5",
					Description: "Ensure that default service accounts are not actively used",
				},
				{
					ID:          "5_1_6",
					Name:        "5.1.6",
					Description: "Ensure that Service Account Tokens are only mounted where necessary",
				},
			},
		},
		{
			ID:          "5_2",
			Name:        "5.2",
			Description: "Policies - Pod Security Policies",
			Controls: []Control{
				{
					ID:          "5_2_1",
					Name:        "5.2.1",
					Description: "Minimize the admission of privileged containers",
				},
				{
					ID:          "5_2_2",
					Name:        "5.2.2",
					Description: "Minimize the admission of containers wishing to share the host process ID namespace",
				},
				{
					ID:          "5_2_3",
					Name:        "5.2.3",
					Description: "Minimize the admission of containers wishing to share the host IPC namespace",
				},
				{
					ID:          "5_2_4",
					Name:        "5.2.4",
					Description: "Minimize the admission of containers wishing to share the host network namespace",
				},
				{
					ID:          "5_2_5",
					Name:        "5.2.5",
					Description: "Minimize the admission of containers with allowPrivilegeEscalation",
				},
				{
					ID:          "5_2_6",
					Name:        "5.2.6",
					Description: "Minimize the admission of root containers",
				},
				{
					ID:          "5_2_7",
					Name:        "5.2.7",
					Description: "Minimize the admission of containers with the NET_RAW capability",
				},
				{
					ID:          "5_2_8",
					Name:        "5.2.8",
					Description: "Minimize the admission of containers with added capabilities",
				},
				{
					ID:          "5_2_9",
					Name:        "5.2.9",
					Description: "Minimize the admission of containers with capabilities assigned",
				},
			},
		},
		{
			ID:          "5_3",
			Name:        "5.3",
			Description: "Policies - Network Policies and CNI",
			Controls: []Control{
				{
					ID:          "5_3_1",
					Name:        "5.3.1",
					Description: "Ensure that the CNI in use supports Network Policies",
				},
				{
					ID:          "5_3_2",
					Name:        "5.3.2",
					Description: "Ensure that all Namespaces have Network Policies defined",
				},
			},
		},
		{
			ID:          "5_4",
			Name:        "5.4",
			Description: "Policies - Secrets Management",
			Controls: []Control{
				{
					ID:          "5_4_1",
					Name:        "5.4.1",
					Description: "Prefer using secrets as files over secrets as environment variables",
				},
				{
					ID:          "5_4_2",
					Name:        "5.4.2",
					Description: "Consider external secret storage",
				},
			},
		},
		{
			ID:          "5_5",
			Name:        "5.5",
			Description: "Policies - Extensible Admission Control",
			Controls: []Control{
				{
					ID:          "5_5_1",
					Name:        "5.5.1",
					Description: "Configure Image Provenance using ImagePolicyWebhook admission controller",
				},
			},
		},
		{
			ID:          "5_6",
			Name:        "5.6",
			Description: "Policies - General Policies",
			Controls: []Control{
				{
					ID:          "5_6_1",
					Name:        "5.6.1",
					Description: "Create administrative boundaries between resources using namespaces",
				},
				{
					ID:          "5_6_2",
					Name:        "5.6.2",
					Description: "Ensure that the seccomp profile is set to docker/default in your pod definitions",
				},
				{
					ID:          "5_6_3",
					Name:        "5.6.3",
					Description: "Apply Security Context to Your Pods and Containers",
				},
				{
					ID:          "5_6_4",
					Name:        "5.6.4",
					Description: "The default namespace should not be used",
				},
			},
		},
	},
}

func init() {
	AllStandards = append(AllStandards, cisKubernetes)
}
