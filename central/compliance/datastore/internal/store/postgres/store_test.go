package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAggregationCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := mocks.NewMockDB(mockCtrl)
	store := NewStore(db)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	singleClusterCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance),
			sac.ClusterScopeKeys("validClusterID")))

	t.Run("empty cache returns no data", func(t *testing.T) {
		result, sources, domainMap, err := store.GetAggregationResult(
			ctx,
			"query",
			[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			storage.ComplianceAggregation_STANDARD,
		)
		assert.NoError(t, err)
		assert.Nil(t, result)
		assert.Nil(t, sources)
		assert.Nil(t, domainMap)
	})

	t.Run("panic when no SAC in context", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = store.StoreAggregationResult(context.Background(),
				"query",
				nil,
				storage.ComplianceAggregation_STANDARD,
				nil,
				nil,
				nil,
			)
		})
	})

	t.Run("store data", func(t *testing.T) {
		err := store.StoreAggregationResult(
			ctx,
			"query",
			[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CLUSTER},
			storage.ComplianceAggregation_STANDARD,
			[]*storage.ComplianceAggregation_Result{},
			[]*storage.ComplianceAggregation_Source{},
			map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain{},
		)
		assert.NoError(t, err)
	})

	t.Run("read stored data", func(t *testing.T) {
		result, sources, domainMap, err := store.GetAggregationResult(
			ctx,
			"query",
			[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER, storage.ComplianceAggregation_NODE},
			storage.ComplianceAggregation_STANDARD,
		)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, sources)
		assert.NotNil(t, domainMap)
	})

	t.Run("read stored data with different access returns nils", func(t *testing.T) {
		result, sources, domainMap, err := store.GetAggregationResult(
			singleClusterCtx,
			"query",
			[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CLUSTER},
			storage.ComplianceAggregation_STANDARD,
		)
		assert.NoError(t, err)
		assert.Nil(t, result)
		assert.Nil(t, sources)
		assert.Nil(t, domainMap)
	})

	t.Run("clear cache return no error", func(t *testing.T) {
		err := store.ClearAggregationResults(context.Background())
		assert.NoError(t, err)
	})

	t.Run("cleared cache returns no data", func(t *testing.T) {
		result, sources, domainMap, err := store.GetAggregationResult(
			ctx,
			"query",
			[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			storage.ComplianceAggregation_STANDARD,
		)
		assert.NoError(t, err)
		assert.Nil(t, result)
		assert.Nil(t, sources)
		assert.Nil(t, domainMap)
	})
}
