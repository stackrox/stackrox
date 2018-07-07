package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeCluster(t *testing.T) {
	cases := []struct {
		name     string
		cluster  *v1.Cluster
		expected string
	}{
		{
			name: "Happy path",
			cluster: &v1.Cluster{
				CentralApiEndpoint: "localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "http",
			cluster: &v1.Cluster{
				CentralApiEndpoint: "http://localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "https",
			cluster: &v1.Cluster{
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
		cluster       *v1.Cluster
		expectedError bool
	}{
		{
			name:          "Empty Cluster",
			cluster:       &v1.Cluster{},
			expectedError: true,
		},
		{
			name: "No name",
			cluster: &v1.Cluster{
				PreventImage:       "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Image",
			cluster: &v1.Cluster{
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Central Endpoint",
			cluster: &v1.Cluster{
				Name:         "name",
				PreventImage: "image",
			},
			expectedError: true,
		},
		{
			name: "Central Endpoint w/o port",
			cluster: &v1.Cluster{
				Name:               "name",
				PreventImage:       "image",
				CentralApiEndpoint: "central",
			},
			expectedError: true,
		},
		{
			name: "Happy path",
			cluster: &v1.Cluster{
				Name:               "name",
				PreventImage:       "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectedError, validateInput(c.cluster) != nil)
		})
	}

}
