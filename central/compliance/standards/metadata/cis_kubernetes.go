package metadata

var cisKubernetes = Standard{
	ID:   "CIS_Kubernetes_v1_2_0",
	Name: "CIS Kubernetes v1.2.0",
	Categories: []Category{
		{
			ID:          "1_1",
			Name:        "1.1",
			Description: "Master Node Security Configuration - API Server",
			Controls: []Control{
				{
					ID:          "1_1_1",
					Name:        "1.1.1",
					Description: "Ensure that the --anonymous-auth argument is set to false",
				},
				{
					ID:          "1_1_2",
					Name:        "1.1.2",
					Description: "Ensure that the --basic-auth-file argument is not set",
				},
				{
					ID:          "1_1_3",
					Name:        "1.1.3",
					Description: "Ensure that the --insecure-allow-any-token argument is not set",
				},
				{
					ID:          "1_1_4",
					Name:        "1.1.4",
					Description: "Ensure that the --kubelet-https argument is set to true",
				},
				{
					ID:          "1_1_5",
					Name:        "1.1.5",
					Description: "Ensure that the --insecure-bind-address argument is not set",
				},
				{
					ID:          "1_1_6",
					Name:        "1.1.6",
					Description: "Ensure that the --insecure-port argument is set to 0",
				},
				{
					ID:          "1_1_7",
					Name:        "1.1.7",
					Description: "Ensure that the --secure-port argument is not set to 0",
				},
				{
					ID:          "1_1_8",
					Name:        "1.1.8",
					Description: "Ensure that the --profiling argument is set to false",
				},
				{
					ID:          "1_1_9",
					Name:        "1.1.9",
					Description: "Ensure that the --repair-malformed-updates argument is set to false",
				},
				{
					ID:          "1_1_10",
					Name:        "1.1.10",
					Description: "Ensure that the admission control policy is not set to AlwaysAdmit",
				},
				{
					ID:          "1_1_11",
					Name:        "1.1.11",
					Description: "Ensure that the admission control policy is set to AlwaysPullImages",
				},
				{
					ID:          "1_1_12",
					Name:        "1.1.12",
					Description: "Ensure that the admission control policy is set to DenyEscalatingExec",
				},
				{
					ID:          "1_1_13",
					Name:        "1.1.13",
					Description: "Ensure that the admission control policy is set to SecurityContextDeny",
				},
				{
					ID:          "1_1_14",
					Name:        "1.1.14",
					Description: "Ensure that the admission control policy is set to NamespaceLifecycle",
				},
				{
					ID:          "1_1_15",
					Name:        "1.1.15",
					Description: "Ensure that the --audit-log-path argument is set as appropriate",
				},
				{
					ID:          "1_1_16",
					Name:        "1.1.16",
					Description: "Ensure that the --audit-log-maxage argument is set to 30 or as appropriate",
				},
				{
					ID:          "1_1_17",
					Name:        "1.1.17",
					Description: "Ensure that the --audit-log-maxbackup argument is set to 10 or as appropriate",
				},
				{
					ID:          "1_1_18",
					Name:        "1.1.18",
					Description: "Ensure that the --audit-log-maxsize argument is set to 100 or as appropriate",
				},
				{
					ID:          "1_1_19",
					Name:        "1.1.19",
					Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",
				},
				{
					ID:          "1_1_20",
					Name:        "1.1.20",
					Description: "Ensure that the --token-auth-file parameter is not set",
				},
				{
					ID:          "1_1_21",
					Name:        "1.1.21",
					Description: "Ensure that the --kubelet-certificate-authority argument is set as appropriate",
				},
				{
					ID:          "1_1_22",
					Name:        "1.1.22",
					Description: "Ensure that the --kubelet-client-certificate and --kubelet-client-key arguments are set as appropriate",
				},
				{
					ID:          "1_1_23",
					Name:        "1.1.23",
					Description: "Ensure that the --service-account-lookup argument is set to true",
				},
				{
					ID:          "1_1_24",
					Name:        "1.1.24",
					Description: "Ensure that the admission control policy is set to PodSecurityPolicy",
				},
				{
					ID:          "1_1_25",
					Name:        "1.1.25",
					Description: "Ensure that the --service-account-key-file argument is set as appropriate",
				},
				{
					ID:          "1_1_26",
					Name:        "1.1.26",
					Description: "Ensure that the --etcd-certfile and --etcd-keyfile arguments are set as appropriate",
				},
				{
					ID:          "1_1_27",
					Name:        "1.1.27",
					Description: "Ensure that the admission control policy is set to ServiceAccount",
				},
				{
					ID:          "1_1_28",
					Name:        "1.1.28",
					Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
				},
				{
					ID:          "1_1_29",
					Name:        "1.1.29",
					Description: "Ensure that the --client-ca-file argument is set as appropriate",
				},
				{
					ID:          "1_1_30",
					Name:        "1.1.30",
					Description: "Ensure that the --etcd-cafile argument is set as appropriate",
				},
				{
					ID:          "1_1_31",
					Name:        "1.1.31",
					Description: "Ensure that the --authorization-mode argument is set to Node",
				},
				{
					ID:          "1_1_32",
					Name:        "1.1.32",
					Description: "Ensure that the admission control policy is set to NodeRestriction",
				},
				{
					ID:          "1_1_33",
					Name:        "1.1.33",
					Description: "Ensure that the --experimental-encryption-provider-config argument is set as appropriate",
				},
				{
					ID:          "1_1_34",
					Name:        "1.1.34",
					Description: "Ensure that the encryption provider is set to aescbc",
				},
				{
					ID:          "1_1_35",
					Name:        "1.1.35",
					Description: "Ensure that the admission control policy is set to EventRateLimit",
				},
				{
					ID:          "1_1_36",
					Name:        "1.1.36",
					Description: "Ensure that the AdvancedAuditing argument is not set to false",
				},
				{
					ID:          "1_1_37",
					Name:        "1.1.37",
					Description: "Ensure that the --request-timeout argument is set as appropriate",
				},
			},
		},
		{
			ID:          "1_2",
			Name:        "1.2",
			Description: "Master Node Security Configuration - Scheduler",
			Controls: []Control{
				{
					ID:          "1_2_1",
					Name:        "1.2.1",
					Description: "Ensure that the --profiling argument is set to false",
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
					Description: "Apply Security Context to Your Pods and Containers",
				},
				{
					ID:          "1_3_7",
					Name:        "1.3.7",
					Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",
				},
			},
		},
		{
			ID:          "1_4",
			Name:        "1.4",
			Description: "Master Node Security Configuration - Configuration Files",
			Controls: []Control{
				{
					ID:          "1_4_1",
					Name:        "1.4.1",
					Description: "Ensure that the API server pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_2",
					Name:        "1.4.2",
					Description: "Ensure that the API server pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_4_3",
					Name:        "1.4.3",
					Description: "Ensure that the controller manager pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_4",
					Name:        "1.4.4",
					Description: "Ensure that the controller manager pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_4_5",
					Name:        "1.4.5",
					Description: "Ensure that the scheduler pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_6",
					Name:        "1.4.6",
					Description: "Ensure that the scheduler pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_4_7",
					Name:        "1.4.7",
					Description: "Ensure that the etcd pod specification file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_8",
					Name:        "1.4.8",
					Description: "Ensure that the etcd pod specification file ownership is set to root:root",
				},
				{
					ID:          "1_4_9",
					Name:        "1.4.9",
					Description: "Ensure that the Container Network Interface file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_10",
					Name:        "1.4.10",
					Description: "Ensure that the Container Network Interface file ownership is set to root:root",
				},
				{
					ID:          "1_4_11",
					Name:        "1.4.11",
					Description: "Ensure that the etcd data directory permissions are set to 700 or more restrictive",
				},
				{
					ID:          "1_4_12",
					Name:        "1.4.12",
					Description: "Ensure that the etcd data directory ownership is set to etcd:etcd",
				},
				{
					ID:          "1_4_13",
					Name:        "1.4.13",
					Description: "Ensure that the admin.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_14",
					Name:        "1.4.14",
					Description: "Ensure that the admin.conf file ownership is set to root:root",
				},
				{
					ID:          "1_4_15",
					Name:        "1.4.15",
					Description: "Ensure that the scheduler.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_16",
					Name:        "1.4.16",
					Description: "Ensure that the scheduler.conf file ownership is set to root:root",
				},
				{
					ID:          "1_4_17",
					Name:        "1.4.17",
					Description: "Ensure that the controller-manager.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "1_4_18",
					Name:        "1.4.18",
					Description: "Ensure that the controller-manager.conf file ownership is set to root:root",
				},
			},
		},
		{
			ID:          "1_5",
			Name:        "1.5",
			Description: "Master Node Security Configuration - etcd",
			Controls: []Control{
				{
					ID:          "1_5_1",
					Name:        "1.5.1",
					Description: "Ensure that the --cert-file and --key-file arguments are set as appropriate",
				},
				{
					ID:          "1_5_2",
					Name:        "1.5.2",
					Description: "Ensure that the --client-cert-auth argument is set to true",
				},
				{
					ID:          "1_5_3",
					Name:        "1.5.3",
					Description: "Ensure that the --auto-tls argument is not set to true",
				},
				{
					ID:          "1_5_4",
					Name:        "1.5.4",
					Description: "Ensure that the --peer-cert-file and --peer-key-file arguments are set as appropriate",
				},
				{
					ID:          "1_5_5",
					Name:        "1.5.5",
					Description: "Ensure that the --peer-client-cert-auth argument is set to true",
				},
				{
					ID:          "1_5_6",
					Name:        "1.5.6",
					Description: "Ensure that the --peer-auto-tls argument is not set to true",
				},
				{
					ID:          "1_5_7",
					Name:        "1.5.7",
					Description: "Ensure that the --wal-dir argument is set as appropriate",
				},
				{
					ID:          "1_5_8",
					Name:        "1.5.8",
					Description: "Ensure that the --max-wals argument is set to 0",
				},
				{
					ID:          "1_5_9",
					Name:        "1.5.9",
					Description: "Ensure that a unique Certificate Authority is used for etcd",
				},
			},
		},
		{
			ID:          "1_6",
			Name:        "1.6",
			Description: "Master Node Security Configuration - General Security Primitives",
			Controls: []Control{
				{
					ID:          "1_6_1",
					Name:        "1.6.1",
					Description: "Ensure that the cluster-admin role is only used where required",
				},
				{
					ID:          "1_6_2",
					Name:        "1.6.2",
					Description: "Create Pod Security Policies for your cluster",
				},
				{
					ID:          "1_6_3",
					Name:        "1.6.3",
					Description: "Create administrative boundaries between resources using namespaces",
				},
				{
					ID:          "1_6_4",
					Name:        "1.6.4",
					Description: "Create network segmentation using Network Policies",
				},
				{
					ID:          "1_6_5",
					Name:        "1.6.5",
					Description: "Ensure that the seccomp profile is set to docker/default in your pod definitions",
				},
				{
					ID:          "1_6_6",
					Name:        "1.6.6",
					Description: "Apply Security Context to Your Pods and Containers",
				},
				{
					ID:          "1_6_7",
					Name:        "1.6.7",
					Description: "Configure Image Provenance using ImagePolicyWebhook admission controller",
				},
				{
					ID:          "1_6_8",
					Name:        "1.6.8",
					Description: "Configure Network policies as appropriate",
				},
				{
					ID:          "1_6_9",
					Name:        "1.6.9",
					Description: "Place compensating controls in the form of PSP and RBAC for privileged containers usage",
				},
			},
		},
		{
			ID:          "2_1",
			Name:        "2.1",
			Description: "Worker Node Security Configuration - Kubelet",
			Controls: []Control{
				{
					ID:          "2_1_1",
					Name:        "2.1.1",
					Description: "Ensure that the --allow-privileged argument is set to false",
				},
				{
					ID:          "2_1_2",
					Name:        "2.1.2",
					Description: "Ensure that the --anonymous-auth argument is set to false",
				},
				{
					ID:          "2_1_3",
					Name:        "2.1.3",
					Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",
				},
				{
					ID:          "2_1_4",
					Name:        "2.1.4",
					Description: "Ensure that the --client-ca-file argument is set as appropriate",
				},
				{
					ID:          "2_1_5",
					Name:        "2.1.5",
					Description: "Ensure that the --read-only-port argument is set to 0",
				},
				{
					ID:          "2_1_6",
					Name:        "2.1.6",
					Description: "Ensure that the --streaming-connection-idle-timeout argument is not set to 0",
				},
				{
					ID:          "2_1_7",
					Name:        "2.1.7",
					Description: "Ensure that the --protect-kernel-defaults argument is set to true",
				},
				{
					ID:          "2_1_8",
					Name:        "2.1.8",
					Description: "Ensure that the --make-iptables-util-chains argument is set to true",
				},
				{
					ID:          "2_1_9",
					Name:        "2.1.9",
					Description: "Ensure that the --keep-terminated-pod-volumes argument is set to false",
				},
				{
					ID:          "2_1_10",
					Name:        "2.1.10",
					Description: "Ensure that the --hostname-override argument is not set",
				},
				{
					ID:          "2_1_11",
					Name:        "2.1.11",
					Description: "Ensure that the --event-qps argument is set to 0",
				},
				{
					ID:          "2_1_12",
					Name:        "2.1.12",
					Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
				},
				{
					ID:          "2_1_13",
					Name:        "2.1.13",
					Description: "Ensure that the --cadvisor-port argument is set to 0",
				},
				{
					ID:          "2_1_14",
					Name:        "2.1.14",
					Description: "Ensure that the RotateKubeletClientCertificate argument is not set to false",
				},
				{
					ID:          "2_1_15",
					Name:        "2.1.15",
					Description: "Ensure that the RotateKubeletServerCertificate argument is set to true",
				},
			},
		},
		{
			ID:          "2_2",
			Name:        "2.2",
			Description: "Worker Node Security Configuration - Configuration Files",
			Controls: []Control{
				{
					ID:          "2_2_1",
					Name:        "2.2.1",
					Description: "Ensure that the kubelet.conf file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "2_2_2",
					Name:        "2.2.2",
					Description: "Ensure that the kubelet.conf file ownership is set to root:root",
				},
				{
					ID:          "2_2_3",
					Name:        "2.2.3",
					Description: "Ensure that the kubelet service file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "2_2_4",
					Name:        "2.2.4",
					Description: "Ensure that the kubelet service file ownership is set to root:root",
				},
				{
					ID:          "2_2_5",
					Name:        "2.2.5",
					Description: "Ensure that the proxy kubeconfig file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "2_2_6",
					Name:        "2.2.6",
					Description: "Ensure that the proxy kubeconfig file ownership is set to root:root",
				},
				{
					ID:          "2_2_7",
					Name:        "2.2.7",
					Description: "Ensure that the certificate authorities file permissions are set to 644 or more restrictive",
				},
				{
					ID:          "2_2_8",
					Name:        "2.2.8",
					Description: "Ensure that the client certificate authorities file ownership is set to root:root",
				},
			},
		},
		{
			ID:          "3_1",
			Name:        "3.1",
			Description: "Federated Deployments - Federation API Server",
			Controls: []Control{
				{
					ID:          "3_1_1",
					Name:        "3.1.1",
					Description: "Ensure that the --anonymous-auth argument is set to false",
				},
				{
					ID:          "3_1_2",
					Name:        "3.1.2",
					Description: "Ensure that the --basic-auth-file argument is not set",
				},
				{
					ID:          "3_1_3",
					Name:        "3.1.3",
					Description: "Ensure that the --insecure-allow-any-token argument is not set",
				},
				{
					ID:          "3_1_4",
					Name:        "3.1.4",
					Description: "Ensure that the --insecure-bind-address argument is not set",
				},
				{
					ID:          "3_1_5",
					Name:        "3.1.5",
					Description: "Ensure that the --insecure-port argument is set to 0",
				},
				{
					ID:          "3_1_6",
					Name:        "3.1.6",
					Description: "Ensure that the --secure-port argument is not set to 0",
				},
				{
					ID:          "3_1_7",
					Name:        "3.1.7",
					Description: "Ensure that the --profiling argument is set to false",
				},
				{
					ID:          "3_1_8",
					Name:        "3.1.8",
					Description: "Ensure that the admission control policy is not set to AlwaysAdmit",
				},
				{
					ID:          "3_1_9",
					Name:        "3.1.9",
					Description: "Ensure that the admission control policy is set to NamespaceLifecycle",
				},
				{
					ID:          "3_1_10",
					Name:        "3.1.10",
					Description: "Ensure that the --audit-log-path argument is set as appropriate",
				},
				{
					ID:          "3_1_11",
					Name:        "3.1.11",
					Description: "Ensure that the --audit-log-maxage argument is set to 30 or as appropriate",
				},
				{
					ID:          "3_1_12",
					Name:        "3.1.12",
					Description: "Ensure that the --audit-log-maxbackup argument is set to 10 or as appropriate",
				},
				{
					ID:          "3_1_13",
					Name:        "3.1.13",
					Description: "Ensure that the --audit-log-maxsize argument is set to 100 or as appropriate",
				},
				{
					ID:          "3_1_14",
					Name:        "3.1.14",
					Description: "Ensure that the --authorization-mode argument is not set to AlwaysAllow",
				},
				{
					ID:          "3_1_15",
					Name:        "3.1.15",
					Description: "Ensure that the --token-auth-file parameter is not set",
				},
				{
					ID:          "3_1_16",
					Name:        "3.1.16",
					Description: "Ensure that the --service-account-lookup argument is set to true",
				},
				{
					ID:          "3_1_17",
					Name:        "3.1.17",
					Description: "Ensure that the --service-account-key-file argument is set as appropriate",
				},
				{
					ID:          "3_1_18",
					Name:        "3.1.18",
					Description: "Ensure that the --etcd-certfile and --etcd-keyfile arguments are set as appropriate",
				},
				{
					ID:          "3_1_19",
					Name:        "3.1.19",
					Description: "Ensure that the --tls-cert-file and --tls-private-key-file arguments are set as appropriate",
				},
			},
		},
		{
			ID:          "3_2",
			Name:        "3.2",
			Description: "Federated Deployments - Federation Controller Manager",
			Controls: []Control{
				{
					ID:          "3_2_1",
					Name:        "3.2.1",
					Description: "Ensure that the --profiling argument is set to false",
				},
			},
		},
	},
}

func init() {
	AllStandards = append(AllStandards, cisKubernetes)
}
