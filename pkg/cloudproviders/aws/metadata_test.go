package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetClusterMetadataFromNodeLabels(t *testing.T) {

	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()
	expectedClusterName := "my-cluster"
	expectedClusterID := "arn:aws:eks:us-east-1:1234:cluster/my-cluster"

	_, err := k8sClient.CoreV1().Nodes().Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-node",
			Labels: map[string]string{eksClusterNameLabel: "my-cluster"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	doc := &imds.InstanceIdentityDocument{
		Region:    "us-east-1",
		AccountID: "1234",
	}
	clusterName, err := getClusterNameFromNodeLabels(ctx, k8sClient)
	require.NoError(t, err)
	clusterMetadata := clusterMetadataFromName(clusterName, doc)
	assert.Equal(t, storage.ClusterMetadata_EKS, clusterMetadata.GetType())
	assert.Equal(t, expectedClusterName, clusterMetadata.GetName())
	assert.Equal(t, expectedClusterID, clusterMetadata.GetId())
}
