package cluster

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var validCluster = &storage.Cluster{
	Name:               "cluster-name",
	MainImage:          "stackrox.io/main:3.0.55.0",
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
			expectedErrors: []string{"invalid main image 'invalid image': invalid reference format"},
		},
		"Cluster with empty main image should not fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.MainImage = ""
			},
		},
		"Cluster with configured collector image tag should fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = "docker.io/stackrox/collector:3.2.0-slim"
			},
			expectedErrors: []string{"collector image may not specify a tag.  Please remove tag '3.2.0-slim' to continue"},
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
			expectedErrors: []string{"invalid collector image 'invalid image': invalid reference format"},
		},
		"Cluster with empty collector image should not fail": {
			configureClusterFn: func(cluster *storage.Cluster) {
				cluster.CollectorImage = ""
			},
		},
		"Helm Managed cluster with configured collector image tag is allowed": {
			configureClusterFn: func(cluster *storage.Cluster) {
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
			cluster := validCluster.CloneVT()
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
			cluster := validCluster.CloneVT()
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

func TestIsNameValid(t *testing.T) {
	shouldPass := []string{
		"a1234",
		"foo",
		"12324.example.com",
		"e123",
		"foo_bar",
		"foo-bar_baz-boom",
		"foo______bar",
	}
	for _, tt := range shouldPass {
		t.Run(fmt.Sprintf("name-%s-should-be-valid", tt), func(t *testing.T) {
			assert.NoError(t, IsNameValid(tt))
		})
	}

	shouldFail := []string{
		"",
		"123",
		"-123",
		"NaN",
		"Inf",
		"0o123",
		"0xFFAA",
		"0b0101010101",
		"1.23e7",
		"0.34",
		"9999998989898989898989898",
		"true",
		"false",
		"null",
		"{}",
		"_foo",
		"-foo",
		"foo-",
	}
	for _, tt := range shouldFail {
		t.Run(fmt.Sprintf("name-%s-should-be-invalid", tt), func(t *testing.T) {
			assert.Error(t, IsNameValid(tt))
		})
	}
}
