package authorizer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeInventoryConcurrentRefresh(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	expected := inventory{clusters: clusters, namespaces: namespaces}
	cache := &scopeInventoryCache{load: func(context.Context) (inventory, error) {
		loads.Add(1)
		close(started)
		<-release
		return expected, nil
	}}

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			result, err := cache.get(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, expected, result)
		})
	}
	<-started
	close(release)
	wg.Wait()
	require.EqualValues(t, 1, loads.Load())
}

func TestScopeInventoryExpiredRefreshFailure(t *testing.T) {
	loadError := errors.New("namespace read failed")
	initial := inventory{clusters: clusters, namespaces: namespaces}
	updated := inventory{clusters: []effectiveaccessscope.Cluster{&storage.Cluster{Id: "new-cluster"}}}
	var loads int
	cache := &scopeInventoryCache{load: func(context.Context) (inventory, error) {
		loads++
		switch loads {
		case 1:
			return initial, nil
		case 2:
			// A successful cluster read followed by a failed namespace read
			// must not replace the snapshot or authorize with expired data.
			return updated, loadError
		default:
			return updated, nil
		}
	}}
	result, err := cache.get(context.Background())
	require.NoError(t, err)
	require.Equal(t, initial, result)
	result, err = cache.get(context.Background())
	require.NoError(t, err)
	require.Equal(t, initial, result)
	require.Equal(t, 1, loads)

	cache.refreshedAt = time.Now().Add(-scopeInventoryLifetime)
	result, err = cache.get(context.Background())
	require.ErrorIs(t, err, loadError)
	require.Empty(t, result)
	require.Equal(t, initial, cache.inventory)

	result, err = cache.get(context.Background())
	require.NoError(t, err)
	require.Equal(t, updated, result)
	require.Equal(t, 3, loads)
}

func TestScopeInventoryCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	cache := &scopeInventoryCache{load: func(context.Context) (inventory, error) {
		close(started)
		<-release
		return inventory{}, nil
	}}
	var wg sync.WaitGroup
	wg.Go(func() {
		_, err := cache.get(context.Background())
		assert.NoError(t, err)
	})
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := cache.get(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	close(release)
	wg.Wait()

	// Even a cache hit must respect a canceled request.
	_, err = cache.get(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestScopeInventoryLabelRevocation(t *testing.T) {
	namespace := &storage.NamespaceMetadata{
		Id: "namespace", Name: firstNamespaceName, ClusterId: firstClusterID,
		ClusterName: firstClusterName, Labels: map[string]string{"team": "allowed"},
	}
	cache := &scopeInventoryCache{load: func(context.Context) (inventory, error) {
		return inventory{clusters: clusters, namespaces: []effectiveaccessscope.Namespace{namespace.CloneVT()}}, nil
	}}
	roles := []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{
		Id: "label-scope",
		Rules: &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
			Requirements: []*storage.SetBasedLabelSelector_Requirement{{Key: "team", Op: storage.SetBasedLabelSelector_IN, Values: []string{"allowed"}}},
		}}},
	})}
	allowed := func() bool {
		result, err := cache.get(context.Background())
		require.NoError(t, err)
		checker := newGlobalScopeCheckerCore(result.clusters, result.namespaces, roles, nil)
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(), checker)
		return sac.ForResource(resources.Deployment).ScopeChecker(ctx, storage.Access_READ_ACCESS,
			sac.ClusterScopeKey(firstClusterID), sac.NamespaceScopeKey(firstNamespaceName)).IsAllowed()
	}
	require.True(t, allowed())
	namespace.Labels = map[string]string{"team": "other"}
	require.True(t, allowed(), "requests within the snapshot lifetime retain the published inventory")
	cache.refreshedAt = time.Now().Add(-scopeInventoryLifetime)
	require.False(t, allowed(), "new requests must observe revoked label access after refresh")
}
