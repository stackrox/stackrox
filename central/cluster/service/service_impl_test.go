package service

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeCluster(t *testing.T) {
	cases := []struct {
		name     string
		cluster  *storage.Cluster
		expected string
	}{
		{
			name: "Happy path",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "http",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "http://localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "https",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "https://localhost:8080",
			},
			expected: "localhost:8080",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			normalizeCluster(c.cluster)
			assert.Equal(t, c.expected, c.cluster.GetCentralApiEndpoint())
		})
	}
}

func TestValidateCluster(t *testing.T) {
	cases := []struct {
		name          string
		cluster       *storage.Cluster
		expectedError bool
	}{
		{
			name:          "Empty Cluster",
			cluster:       &storage.Cluster{},
			expectedError: true,
		},
		{
			name: "No name",
			cluster: &storage.Cluster{
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Image",
			cluster: &storage.Cluster{
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "Image without tag",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Non-trivial image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main:1.2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Moderately complex image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.512-125125",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Image with SHA",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Invalid image - contains spaces",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.3 injectedCommand",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Central Endpoint",
			cluster: &storage.Cluster{
				Name:      "name",
				MainImage: "image",
			},
			expectedError: true,
		},
		{
			name: "Central Endpoint w/o port",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central",
			},
			expectedError: true,
		},
		{
			name: "K8s Empty Namespace",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				OrchestratorParams: &storage.Cluster_Kubernetes{
					Kubernetes: &storage.KubernetesParams{
						Params: &storage.CommonKubernetesParams{
							Namespace: "",
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "K8s Namespace with spaces",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				OrchestratorParams: &storage.Cluster_Kubernetes{
					Kubernetes: &storage.KubernetesParams{
						Params: &storage.CommonKubernetesParams{
							Namespace: "I HAVE SPACES",
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "OpenShift Namespace with spaces",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				OrchestratorParams: &storage.Cluster_Openshift{
					Openshift: &storage.OpenshiftParams{
						Params: &storage.CommonKubernetesParams{
							Namespace: "I HAVE SPACES",
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Happy path K8s",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				OrchestratorParams: &storage.Cluster_Kubernetes{
					Kubernetes: &storage.KubernetesParams{
						Params: &storage.CommonKubernetesParams{
							Namespace: "valid-dns-name-again",
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Happy path",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateInput(c.cluster)
			if c.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
