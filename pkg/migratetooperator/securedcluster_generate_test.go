package migratetooperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformToSecuredCluster(t *testing.T) {
	src := &fakeSource{}
	cr, warnings, err := TransformToSecuredCluster(src, "my-cluster")
	require.NoError(t, err)
	assert.Empty(t, warnings)

	assert.Equal(t, "platform.stackrox.io/v1alpha1", cr.APIVersion)
	assert.Equal(t, "SecuredCluster", cr.Kind)
	assert.Equal(t, "stackrox-secured-cluster-services", cr.Name)
	require.NotNil(t, cr.Spec.ClusterName)
	assert.Equal(t, "my-cluster", *cr.Spec.ClusterName)
}
