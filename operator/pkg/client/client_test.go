package client

import (
	"context"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	centralv1alpha1 "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	fakeCentral "github.com/stackrox/rox/operator/pkg/clientset/stackrox/fake"
	fakeSecuredCluster "github.com/stackrox/rox/operator/pkg/clientset/stackrox/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespace = "stackrox-test"

func TestClientSet(t *testing.T) {
	//Skip test as the client-gen currently has a bug for generating groups, see: https://github.com/kubernetes/kubernetes/pull/100738
	//TODO(ROX-7628): upgrade to client-gen 1.22 after the PR above was merged
	t.Skip()
	centralFake := fakeCentral.NewSimpleClientset(&centralv1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-test",
			Namespace: namespace,
		},
		Spec: centralv1alpha1.CentralSpec{},
	})

	securedClusterFake := fakeSecuredCluster.NewSimpleClientset(&v1alpha1.SecuredCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secured-cluster-test",
			Namespace: namespace,
		},
		Spec: v1alpha1.SecuredClusterSpec{},
	})

	client := StackRoxClientset{
		centralClientSet:        centralFake,
		securedClusterClientSet: securedClusterFake,
	}

	centralResult, err := client.CentralV1Alpha1(namespace).Get(context.TODO(), "central-test", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "central-test", centralResult.GetName())

	securedClusterResult, err := client.SecuredClusterV1Alpha1(namespace).Get(context.TODO(), "secured-cluster-test", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "secured-cluster-test", securedClusterResult.GetName())
}
