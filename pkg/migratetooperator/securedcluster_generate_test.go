package migratetooperator

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformToSecuredCluster(t *testing.T) {
	src := &securedClusterFakeSource{clusterName: "my-cluster"}
	cr, warnings, err := TransformToSecuredCluster(src)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	assert.Equal(t, "platform.stackrox.io/v1alpha1", cr.APIVersion)
	assert.Equal(t, "SecuredCluster", cr.Kind)
	assert.Equal(t, "stackrox-secured-cluster-services", cr.Name)
	require.NotNil(t, cr.Spec.ClusterName)
	assert.Equal(t, "my-cluster", *cr.Spec.ClusterName)
}

func TestTransformToSecuredCluster_MissingSecret(t *testing.T) {
	src := &securedClusterFakeSource{}
	_, _, err := TransformToSecuredCluster(src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

type securedClusterFakeSource struct {
	fakeSource
	clusterName string
}

func (f *securedClusterFakeSource) Secret(name string) (*corev1.Secret, error) {
	if name == "helm-effective-cluster-name" && f.clusterName != "" {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			StringData: map[string]string{"cluster-name": f.clusterName},
		}, nil
	}
	return nil, nil
}
