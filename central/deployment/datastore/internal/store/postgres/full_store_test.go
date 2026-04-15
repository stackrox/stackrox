//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetListDeployment tests the GetListDeployment method from the full store.
func TestGetListDeployment(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	fullStore := NewFullStore(testDB.DB)

	dep := &storage.StoredDeployment{
		Id:          uuid.NewV4().String(),
		Hash:        1234567890,
		Name:        "test-deployment",
		ClusterId:   "cluster-id-1",
		ClusterName: "test-cluster",
		Namespace:   "test-namespace",
		NamespaceId: "cluster-id-1test-namespace",
	}
	require.NoError(t, testutils.FullInit(dep, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	require.NoError(t, fullStore.Upsert(ctx, dep))

	t.Run("Get existing deployment", func(t *testing.T) {
		result, found, err := fullStore.GetListDeployment(ctx, dep.GetId())
		assert.NoError(t, err)
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, dep.GetId(), result.GetId())
		assert.Equal(t, dep.GetHash(), result.GetHash())
		assert.Equal(t, dep.GetName(), result.GetName())
		assert.Equal(t, dep.GetClusterName(), result.GetCluster())
		assert.Equal(t, dep.GetClusterId(), result.GetClusterId())
		assert.Equal(t, dep.GetNamespace(), result.GetNamespace())
	})

	t.Run("Get non-existent deployment", func(t *testing.T) {
		nonExistentID := uuid.NewV4().String()
		result, found, err := fullStore.GetListDeployment(ctx, nonExistentID)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, result)
	})
}

// TestGetManyListDeployments tests the GetManyListDeployments method from the full store.
func TestGetManyListDeployments(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	fullStore := NewFullStore(testDB.DB)

	// Create test deployments
	dep1 := &storage.StoredDeployment{
		Id:          uuid.NewV4().String(),
		Hash:        1111111111,
		Name:        "deployment-1",
		ClusterId:   "cluster-1",
		ClusterName: "cluster-1-name",
		Namespace:   "namespace-1",
		NamespaceId: "cluster-1namespace-1",
	}
	dep2 := &storage.StoredDeployment{
		Id:          uuid.NewV4().String(),
		Hash:        2222222222,
		Name:        "deployment-2",
		ClusterId:   "cluster-1",
		ClusterName: "cluster-1-name",
		Namespace:   "namespace-2",
		NamespaceId: "cluster-1namespace-2",
	}
	dep3 := &storage.StoredDeployment{
		Id:          uuid.NewV4().String(),
		Hash:        3333333333,
		Name:        "deployment-3",
		ClusterId:   "cluster-2",
		ClusterName: "cluster-2-name",
		Namespace:   "namespace-1",
		NamespaceId: "cluster-2namespace-1",
	}

	require.NoError(t, testutils.FullInit(dep1, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	require.NoError(t, testutils.FullInit(dep2, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	require.NoError(t, testutils.FullInit(dep3, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	require.NoError(t, fullStore.Upsert(ctx, dep1))
	require.NoError(t, fullStore.Upsert(ctx, dep2))
	require.NoError(t, fullStore.Upsert(ctx, dep3))

	t.Run("Get all deployments with valid IDs", func(t *testing.T) {
		ids := []string{dep1.GetId(), dep2.GetId(), dep3.GetId()}
		results, missingIndices, err := fullStore.GetManyListDeployments(ctx, ids...)
		assert.NoError(t, err)
		assert.Empty(t, missingIndices)
		assert.Len(t, results, 3)

		// Results should be in same order as input IDs
		assert.Equal(t, dep1.GetId(), results[0].GetId())
		assert.Equal(t, dep2.GetId(), results[1].GetId())
		assert.Equal(t, dep3.GetId(), results[2].GetId())
	})

	t.Run("Get deployments with some missing IDs", func(t *testing.T) {
		nonExistentID1 := uuid.NewV4().String()
		nonExistentID2 := uuid.NewV4().String()

		ids := []string{dep1.GetId(), nonExistentID1, dep2.GetId(), nonExistentID2, dep3.GetId()}
		results, missingIndices, err := fullStore.GetManyListDeployments(ctx, ids...)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, []int{1, 3}, missingIndices)

		// Results should preserve order
		assert.Equal(t, dep1.GetId(), results[0].GetId())
		assert.Equal(t, dep2.GetId(), results[1].GetId())
		assert.Equal(t, dep3.GetId(), results[2].GetId())
	})

	t.Run("Get deployments with all missing IDs", func(t *testing.T) {
		nonExistentID1 := uuid.NewV4().String()
		nonExistentID2 := uuid.NewV4().String()

		ids := []string{nonExistentID1, nonExistentID2}
		results, missingIndices, err := fullStore.GetManyListDeployments(ctx, ids...)
		assert.NoError(t, err)
		assert.Empty(t, results)
		assert.Equal(t, []int{0, 1}, missingIndices)
	})

	t.Run("Order preservation with different request order", func(t *testing.T) {
		ids := []string{dep3.GetId(), dep1.GetId(), dep2.GetId()}
		results, missingIndices, err := fullStore.GetManyListDeployments(ctx, ids...)
		assert.NoError(t, err)
		assert.Empty(t, missingIndices)
		assert.Len(t, results, 3)

		// Results should match requested order
		assert.Equal(t, dep3.GetId(), results[0].GetId())
		assert.Equal(t, dep1.GetId(), results[1].GetId())
		assert.Equal(t, dep2.GetId(), results[2].GetId())
	})

	t.Run("Empty ID list", func(t *testing.T) {
		results, missingIndices, err := fullStore.GetManyListDeployments(ctx)
		assert.NoError(t, err)
		assert.Nil(t, results)
		assert.Nil(t, missingIndices)
	})

	t.Run("Verify ListDeployment fields are populated", func(t *testing.T) {
		ids := []string{dep1.GetId()}
		results, _, err := fullStore.GetManyListDeployments(ctx, ids...)
		assert.NoError(t, err)
		assert.Len(t, results, 1)

		result := results[0]
		assert.Equal(t, dep1.GetId(), result.GetId())
		assert.Equal(t, dep1.GetHash(), result.GetHash())
		assert.Equal(t, dep1.GetName(), result.GetName())
		assert.Equal(t, dep1.GetClusterName(), result.GetCluster())
		assert.Equal(t, dep1.GetClusterId(), result.GetClusterId())
		assert.Equal(t, dep1.GetNamespace(), result.GetNamespace())
	})
}
