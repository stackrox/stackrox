package common

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckRBACEnabled(t *testing.T) {
	testCases := []struct {
		name              string
		cluster           *storage.Cluster
		authorizationMode []string
		expected          bool
	}{
		{
			name: "Available but not enabled",
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"rbac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"ABAC",
			},
			expected: false,
		},
		{
			name: "Enabled but not available", // probably not possible,.
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"abac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"RBAC",
			},
			expected: false,
		},
		{
			name: "Available and enabled",
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"rbac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"Node",
				"RBAC",
			},
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isRBACEnabled(testCase.cluster, testCase.authorizationMode))
		})
	}
}

func TestCheckABACEnabled(t *testing.T) {
	testCases := []struct {
		name              string
		cluster           *storage.Cluster
		authorizationMode []string
		expected          bool
	}{
		{
			name: "Available but not enabled",
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"abac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"RBAC",
			},
			expected: false,
		},
		{
			name: "Enabled but not available", // probably not possible,.
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"rbac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"ABAC",
			},
			expected: false,
		},
		{
			name: "Available and enabled", // probably not possible,.
			cluster: &storage.Cluster{
				Status: &storage.ClusterStatus{
					OrchestratorMetadata: &storage.OrchestratorMetadata{
						ApiVersions: []string{
							"abac.authorization.k8s.io",
						},
					},
				},
			},
			authorizationMode: []string{
				"Node",
				"ABAC",
			},
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isABACEnabled(testCase.cluster, testCase.authorizationMode))
		})
	}
}

func TestGetAuthorizationMode(t *testing.T) {
	testCases := []struct {
		name        string
		deployments map[string]*storage.Deployment
		expected    []string
	}{
		{
			deployments: map[string]*storage.Deployment{
				"dep1": { // Copied sample data
					Id:        "98c9fbef-c5f8-5851-8e46-ee2620e08b3c",
					Name:      "static-kube-apiserver-pods",
					Type:      "StaticPods",
					Namespace: "kube-system",
					Labels: map[string]string{
						"component": "kube-apiserver",
						"tier":      "control-plane",
					},
					PodLabels: map[string]string{
						"component": "kube-apiserver",
						"tier":      "control-plane",
					},
					LabelSelector: &storage.LabelSelector{
						MatchLabels: map[string]string{
							"component": "kube-apiserver",
						},
						Requirements: []*storage.LabelSelector_Requirement{},
					},
					Containers: []*storage.Container{
						{
							Id: "98c9fbef-c5f8-5851-8e46-ee2620e08b3c:kube-apiserver",
							Config: &storage.ContainerConfig{
								Env: []*storage.ContainerConfig_EnvironmentConfig{},
								Command: []string{
									"kube-apiserver",
									"--admission-control=Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota",
									"--allow-privileged=true",
									"--requestheader-group-headers=X-Remote-Group",
									"--requestheader-extra-headers-prefix=X-Remote-Extra-",
									"--requestheader-allowed-names=front-proxy-client",
									"--service-account-key-file=/run/config/pki/sa.pub",
									"--tls-cert-file=/run/config/pki/apiserver.crt",
									"--insecure-port=0",
									"--kubelet-client-certificate=/run/config/pki/apiserver-kubelet-client.crt",
									"--advertise-address=192.168.65.3",
									"--secure-port=6443",
									"--requestheader-client-ca-file=/run/config/pki/front-proxy-ca.crt",
									"--enable-bootstrap-token-auth=true",
									"--requestheader-username-headers=X-Remote-User",
									"--tls-private-key-file=/run/config/pki/apiserver.key",
									"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
									"--client-ca-file=/run/config/pki/ca.crt",
									"--kubelet-client-key=/run/config/pki/apiserver-kubelet-client.key",
									"--proxy-client-cert-file=/run/config/pki/front-proxy-client.crt",
									"--proxy-client-key-file=/run/config/pki/front-proxy-client.key",
									"--service-cluster-ip-range=10.96.0.0/12",
									"--authorization-mode=Node,RBAC",
									"--etcd-servers=https://127.0.0.1:2379",
									"--etcd-cafile=/run/config/pki/etcd/ca.crt",
									"--etcd-certfile=/run/config/pki/apiserver-etcd-client.crt",
									"--etcd-keyfile=/run/config/pki/apiserver-etcd-client.key",
								},
							},
							Name: "kube-apiserver",
						},
					},
				},
			},
			expected: []string{
				"Node",
				"RBAC",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, getAPIServerAuthorizationMode(testCase.deployments))
		})
	}
}
