package cluster

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validCluster = storage.Cluster{
	Name:               "cluster-name",
	MainImage:          "stackrox.io/main:3.0.55.0",
	CentralApiEndpoint: "central.stackrox:443",
	Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
}

func TestValidation(t *testing.T) {
	cluster := validCluster
	errors := Validate(&cluster)
	assert.Nil(t, errors.ToError())
}

func TestValidationWithOpenShift3ClusterAndEnabledControllerWebhookShouldFail(t *testing.T) {
	cluster := validCluster.Clone()
	cluster.AdmissionControllerEvents = true
	cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
	errors := Validate(cluster)

	require.Error(t, errors.ToError())
	assert.Contains(t, errors.String(), "OpenShift 3.x compatibility mode does not support")
}
