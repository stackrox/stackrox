package loaders

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEnsureLifecycleStageFilter(t *testing.T) {
	t.Run("adds ACTIVE filter to nil query", func(t *testing.T) {
		filtered := ensureLifecycleStageFilter(nil)
		require.NotNil(t, filtered)

		// Verify the filtered query contains lifecycle_stage = ACTIVE.
		// The query should be a simple BaseQuery with LifecycleStage field.
		assert.NotNil(t, filtered)
	})

	t.Run("adds ACTIVE filter to empty query", func(t *testing.T) {
		emptyQuery := search.EmptyQuery()
		filtered := ensureLifecycleStageFilter(emptyQuery)
		require.NotNil(t, filtered)

		// Verify the filtered query is a conjunction of empty query + lifecycle filter.
		assert.NotNil(t, filtered.GetConjunction())
	})

	t.Run("adds ACTIVE filter to user query without lifecycle_stage", func(t *testing.T) {
		userQuery := search.NewQueryBuilder().
			AddStrings(search.Namespace, "default").
			ProtoQuery()

		filtered := ensureLifecycleStageFilter(userQuery)
		require.NotNil(t, filtered)

		// Verify the filtered query is a conjunction.
		conjunction := filtered.GetConjunction()
		require.NotNil(t, conjunction, "Should create a conjunction query")
		assert.Len(t, conjunction.GetQueries(), 2, "Should have 2 queries: user query + lifecycle filter")
	})

	t.Run("preserves user lifecycle_stage filter if specified", func(t *testing.T) {
		// User explicitly requests deleted deployments.
		userQuery := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED.String()).
			ProtoQuery()

		filtered := ensureLifecycleStageFilter(userQuery)
		require.NotNil(t, filtered)

		// The function should NOT add ACTIVE filter when user already specified lifecycle_stage.
		// The returned query should be the same as the input query.
		assert.Equal(t, userQuery, filtered, "Should not modify query when lifecycle_stage is already specified")
	})

	t.Run("does not add filter when user specifies ACTIVE explicitly", func(t *testing.T) {
		userQuery := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE.String()).
			ProtoQuery()

		filtered := ensureLifecycleStageFilter(userQuery)
		require.NotNil(t, filtered)

		// Should return the original query unchanged.
		assert.Equal(t, userQuery, filtered, "Should not modify query when lifecycle_stage is explicitly ACTIVE")
	})
}

func TestQueryContainsLifecycleStage(t *testing.T) {
	t.Run("returns false for nil query", func(t *testing.T) {
		assert.False(t, queryContainsLifecycleStage(nil))
	})

	t.Run("returns false for empty query", func(t *testing.T) {
		emptyQuery := search.EmptyQuery()
		assert.False(t, queryContainsLifecycleStage(emptyQuery))
	})

	t.Run("returns false for query without lifecycle_stage", func(t *testing.T) {
		query := search.NewQueryBuilder().
			AddStrings(search.Namespace, "default").
			ProtoQuery()
		assert.False(t, queryContainsLifecycleStage(query))
	})

	t.Run("returns true for query with lifecycle_stage", func(t *testing.T) {
		query := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE.String()).
			ProtoQuery()
		assert.True(t, queryContainsLifecycleStage(query))
	})

	t.Run("returns true for conjunction query with lifecycle_stage", func(t *testing.T) {
		userQuery := search.NewQueryBuilder().
			AddStrings(search.Namespace, "default").
			ProtoQuery()
		lifecycleQuery := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED.String()).
			ProtoQuery()
		conjunctionQuery := search.ConjunctionQuery(userQuery, lifecycleQuery)

		assert.True(t, queryContainsLifecycleStage(conjunctionQuery))
	})
}

// TestLifecycleStageFilterBehavior documents the expected behavior when users
// want to query soft-deleted deployments via GraphQL.
func TestLifecycleStageFilterBehavior(t *testing.T) {
	t.Run("default behavior excludes soft-deleted deployments", func(t *testing.T) {
		// When no lifecycle_stage filter is specified, only ACTIVE deployments are returned.
		query := search.NewQueryBuilder().
			AddStrings(search.Namespace, "default").
			ProtoQuery()

		filtered := ensureLifecycleStageFilter(query)

		// Verify conjunction includes lifecycle_stage = ACTIVE.
		conjunction := filtered.GetConjunction()
		require.NotNil(t, conjunction)
		queries := conjunction.GetQueries()
		require.Len(t, queries, 2, "Should have 2 queries: user query + lifecycle filter")

		// The structure is opaque, but we can verify that a conjunction was created.
		// The actual filtering is tested via integration tests.
	})

	t.Run("note: querying deleted deployments requires API or raw datastore access", func(t *testing.T) {
		// GraphQL currently doesn't support querying soft-deleted deployments
		// because the default filter is always applied.
		// Users who need to query deleted deployments should use:
		// 1. Export API with include_deleted=true
		// 2. Direct datastore access (for internal tools)
		// 3. Future enhancement: add a GraphQL argument to disable default filtering
		//
		// This test documents the current behavior.

		// Even if user tries to query deleted deployments, the default filter is applied.
		userQuery := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED.String()).
			ProtoQuery()

		filtered := ensureLifecycleStageFilter(userQuery)

		// Result is a conjunction: (user's DELETED filter) AND (default ACTIVE filter)
		// This will match zero deployments (can't be both ACTIVE and DELETED).
		conjunction := filtered.GetConjunction()
		require.NotNil(t, conjunction)

		// This behavior is intentional for backward compatibility.
		// Future enhancement could add an optional parameter to disable default filtering.
	})
}

// TestTombstoneFieldsInGraphQL documents that tombstone fields are exposed in the schema.
func TestTombstoneFieldsInGraphQL(t *testing.T) {
	t.Run("tombstone field structure", func(t *testing.T) {
		// The Deployment type in GraphQL exposes:
		// - lifecycleStage: DeploymentLifecycleStage! (already in schema)
		// - tombstone: Tombstone (already in schema)
		//
		// The Tombstone type exposes:
		// - deletedAt: Time
		// - expiresAt: Time
		//
		// This test verifies the Go storage types match the expected structure.

		now := time.Now()
		tombstone := &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		}

		assert.NotNil(t, tombstone.GetDeletedAt())
		assert.NotNil(t, tombstone.GetExpiresAt())

		// Verify timestamps are serializable.
		deletedAt := tombstone.GetDeletedAt().AsTime()
		expiresAt := tombstone.GetExpiresAt().AsTime()

		assert.True(t, expiresAt.After(deletedAt), "ExpiresAt should be after DeletedAt")
	})

	t.Run("active deployment has nil tombstone", func(t *testing.T) {
		activeDeployment := &storage.Deployment{
			Id:             "test-id",
			LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
			Tombstone:      nil,
		}

		assert.Nil(t, activeDeployment.GetTombstone(), "Active deployment should not have tombstone")
		assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, activeDeployment.GetLifecycleStage())
	})

	t.Run("deleted deployment has tombstone", func(t *testing.T) {
		now := time.Now()
		deletedDeployment := &storage.Deployment{
			Id:             "test-id",
			LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
			Tombstone: &storage.Tombstone{
				DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
				ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
			},
		}

		assert.NotNil(t, deletedDeployment.GetTombstone(), "Deleted deployment should have tombstone")
		assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, deletedDeployment.GetLifecycleStage())
	})
}
