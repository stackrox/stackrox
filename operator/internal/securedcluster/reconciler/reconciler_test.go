package reconciler

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCentralEventEnqueuesSecuredClustersAcrossNamespaces(t *testing.T) {
	securedClusters := []*platform.SecuredCluster{
		{ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "central-namespace"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "secured-namespace"}},
	}
	client := testutils.NewFakeClientBuilder(t, securedClusters[0], securedClusters[1]).Build()
	want := []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "local", Namespace: "central-namespace"}},
		{NamespacedName: types.NamespacedName{Name: "remote", Namespace: "secured-namespace"}},
	}

	// The mapping does not depend on whether a Central still exists, so it
	// enqueues both SecuredClusters on both creation and deletion.
	assert.ElementsMatch(t, want, securedClusterRequests(context.Background(), client))
}
