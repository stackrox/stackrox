package migratetooperator

import (
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCGenerate_Default(t *testing.T) {
	config := &securedClusterConfig{clusterName: "test-cluster"}
	cr, warnings := generateSecuredCluster(config)
	assert.Empty(t, warnings)
	assert.Equal(t, "platform.stackrox.io/v1alpha1", cr.APIVersion)
	assert.Equal(t, "SecuredCluster", cr.Kind)
	assert.Equal(t, "stackrox-secured-cluster-services", cr.Name)
	require.NotNil(t, cr.Spec.ClusterName)
	assert.Equal(t, "test-cluster", *cr.Spec.ClusterName)
	assert.Nil(t, cr.Spec.CentralEndpoint)
	assert.Nil(t, cr.Spec.AdmissionControl)
	assert.Nil(t, cr.Spec.PerNode)
}

func TestSCGenerate_CustomCentralEndpoint(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:     "test-cluster",
		centralEndpoint: "my-central.example.com:443",
	}
	cr, _ := generateSecuredCluster(config)
	require.NotNil(t, cr.Spec.CentralEndpoint)
	assert.Equal(t, "my-central.example.com:443", *cr.Spec.CentralEndpoint)
}

func TestSCGenerate_DefaultCentralEndpointOmitted(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:     "test-cluster",
		centralEndpoint: "central.stackrox:443",
	}
	cr, _ := generateSecuredCluster(config)
	assert.Nil(t, cr.Spec.CentralEndpoint)
}

func TestSCGenerate_EnforcementDisabled(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:         "test-cluster",
		enforcementDisabled: true,
	}
	cr, _ := generateSecuredCluster(config)
	require.NotNil(t, cr.Spec.AdmissionControl)
	require.NotNil(t, cr.Spec.AdmissionControl.Enforcement)
	assert.Equal(t, platform.PolicyEnforcementDisabled, *cr.Spec.AdmissionControl.Enforcement)
	assert.Nil(t, cr.Spec.AdmissionControl.FailurePolicy)
}

func TestSCGenerate_FailurePolicyFail(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:       "test-cluster",
		failurePolicyFail: true,
	}
	cr, _ := generateSecuredCluster(config)
	require.NotNil(t, cr.Spec.AdmissionControl)
	require.NotNil(t, cr.Spec.AdmissionControl.FailurePolicy)
	assert.Equal(t, platform.FailurePolicyFail, *cr.Spec.AdmissionControl.FailurePolicy)
}

func TestSCGenerate_CollectionNone(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:    "test-cluster",
		collectionNone: true,
	}
	cr, _ := generateSecuredCluster(config)
	require.NotNil(t, cr.Spec.PerNode)
	require.NotNil(t, cr.Spec.PerNode.Collector)
	require.NotNil(t, cr.Spec.PerNode.Collector.Collection)
	assert.Equal(t, platform.CollectionNone, *cr.Spec.PerNode.Collector.Collection)
}

func TestSCGenerate_TolerationsDisabled(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:         "test-cluster",
		tolerationsDisabled: true,
	}
	cr, _ := generateSecuredCluster(config)
	require.NotNil(t, cr.Spec.PerNode)
	require.NotNil(t, cr.Spec.PerNode.TaintToleration)
	assert.Equal(t, platform.TaintAvoid, *cr.Spec.PerNode.TaintToleration)
}

func TestSCGenerate_CustomImages(t *testing.T) {
	config := &securedClusterConfig{
		clusterName:  "test-cluster",
		customImages: true,
	}
	_, warnings := generateSecuredCluster(config)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "RELATED_IMAGE")
}

func TestSCGenerate_NoWarningsDefault(t *testing.T) {
	config := &securedClusterConfig{clusterName: "test-cluster"}
	_, warnings := generateSecuredCluster(config)
	assert.Empty(t, warnings)
}
