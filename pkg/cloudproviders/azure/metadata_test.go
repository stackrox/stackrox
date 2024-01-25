package azure

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetMetadata_NotOnAzure(t *testing.T) {
	t.Parallel()

	_, err := GetMetadata(context.Background())
	// We might not get metadata info, but we should not get an error.
	assert.NoError(t, err)
}

func TestGetClusterMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()
	expectedClusterName := "my-rg_my-cluster"
	expectedClusterID := "1234_MC_my-rg_my-cluster_eastus"

	_, err := k8sClient.CoreV1().Nodes().Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-node",
			Labels: map[string]string{aksClusterNameLabel: "MC_my-rg_my-cluster_eastus"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	metadata := &azureInstanceMetadata{
		Compute: &computeMetadata{
			Location:       "eastus",
			SubscriptionID: "1234",
		},
	}
	clusterMetadata := getClusterMetadataFromNodeLabels(ctx, k8sClient, metadata)
	assert.Equal(t, storage.ClusterMetadata_AKS, clusterMetadata.GetType())
	assert.Equal(t, expectedClusterName, clusterMetadata.GetName())
	assert.Equal(t, expectedClusterID, clusterMetadata.GetId())
}
