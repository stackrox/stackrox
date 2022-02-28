package cluster

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var validCluster = &storage.Cluster{
	Name:               "cluster-name",
	MainImage:          "stackrox.io/main",
	CentralApiEndpoint: "central.stackrox:443",
	Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
}

func TestPartialValidation(t *testing.T) {
	cases := map[string]struct {
		configureClusterFn func(*storage.Cluster)
		expectedErrors     []string
	}{
		"Valid configureClusterFn validation does not fail": {
			configureClusterFn: func(*storage.Cluster) {},
		},
		"Cluster with invalid main image should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = "invalid image"
			},
			expectedErrors: []string{"invalid image 'invalid image': invalid reference format"},
		},
		"Cluster with empty main image should not fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = ""
			},
		},
		"Cluster with main image with tag should fail when ManagedBy is set to ManagerType_MANAGER_TYPE_UNKNOWN": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = "docker.io/stackrox/main:some_tag"
				// cluster.MainImage = "image"
			},
			expectedErrors: []string{"main image should not contain tags or digests"},
		},
		"Cluster with main image with sha should fail when ManagedBy is set to ManagerType_MANAGER_TYPE_UNKNOWN": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = "docker.io/stackrox/main@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993"
			},
			expectedErrors: []string{"main image should not contain tags or digests"},
		},
		"Cluster with collector image with tag should fail when ManagedBy is set to ManagerType_MANAGER_TYPE_UNKNOWN": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = "docker.io/stackrox/collector:3.2.0-slim"
				cluster.HelmConfig = &storage.CompleteClusterConfig{} // Not really needed since ManagedBy is checked first
			},
			expectedErrors: []string{"collector image should not contain tags or digests"},
		},
		"Cluster with collector image with sha should fail when Managedby is set to ManagerType_MANAGER_TYPE_UNKNOWN": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = "docker.io/stackrox/collector@sha256:8755ac54265892c5aea311e3d73ad771dcbb270d022b1c8cf9cdbf3218b46993"
			},
			expectedErrors: []string{"collector image should not contain tags or digests"},
		},
		"Cluster with configured collector image without tag is valid": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = "docker.io/stackrox/collector"
			},
		},
		"Cluster with invalid collector image should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = "invalid image"
			},
			expectedErrors: []string{"invalid image 'invalid image': invalid reference format"},
		},
		"Cluster with empty collector image should not fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = ""
			},
		},
		"Helm Managed cluster with configured collector image tag is allowed": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.ManagedBy = storage.ManagerType_MANAGER_TYPE_HELM_CHART
				cluster.HelmConfig = &storage.CompleteClusterConfig{}
				cluster.CollectorImage = "docker.io/stackrox/collector:3.2.0-slim"
			},
		},
		"Cluster without central endpoint should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CentralApiEndpoint = ""
			},
			expectedErrors: []string{"Central API Endpoint is required", "Central API Endpoint must be a valid endpoint. Error: empty endpoint specified"},
		},
		"Cluster without central endpoint port fails": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CentralApiEndpoint = "central.stackrox"
			},
			expectedErrors: []string{"Central API Endpoint must have port specified"},
		},
		"Cluster with central endpoint and whitespace should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CentralApiEndpoint = "central. stackrox:443"
			},
			expectedErrors: []string{"Central API endpoint cannot contain whitespace"},
		},
		"OpenShift3 cluster and enabled admission controller webhooks should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.AdmissionControllerEvents = true
				cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
			},
			expectedErrors: []string{"OpenShift 3.x compatibility mode does not support"},
		},
		"Non OpenShift4 cluster with enabled audit log collection should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.DynamicConfig = &storage.DynamicClusterConfig{DisableAuditLogs: false}
				cluster.Type = storage.ClusterType_KUBERNETES_CLUSTER
			},
			expectedErrors: []string{"Audit log collection is only supported on OpenShift 4.x clusters"},
		},
		"OpenShift4 cluster with enabled audit log should be valid": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.DynamicConfig = &storage.DynamicClusterConfig{DisableAuditLogs: false}
				cluster.Type = storage.ClusterType_OPENSHIFT4_CLUSTER
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cluster := validCluster.Clone()
			c.configureClusterFn(cluster)

			gotErrors := ValidatePartial(cluster)

			if len(c.expectedErrors) == 0 {
				assert.NoError(t, gotErrors.ToError(), "expected a valid cluster but received errors")
			}

			for _, expectedErr := range c.expectedErrors {
				assert.Contains(t, gotErrors.String(), expectedErr)
			}
		})
	}
}

func TestFullValidation(t *testing.T) {
	cases := map[string]struct {
		configureClusterFn func(*storage.Cluster)
		expectedErrors     []string
	}{
		"Cluster with empty main image should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = ""
			},
			expectedErrors: []string{"invalid main image '': invalid reference format"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cluster := validCluster.Clone()
			c.configureClusterFn(cluster)

			gotErrors := Validate(cluster)

			if len(c.expectedErrors) == 0 {
				assert.NoError(t, gotErrors.ToError(), "expected a valid cluster but received errors")
			}

			for _, expectedErr := range c.expectedErrors {
				assert.Contains(t, gotErrors.String(), expectedErr)
			}
		})
	}
}
