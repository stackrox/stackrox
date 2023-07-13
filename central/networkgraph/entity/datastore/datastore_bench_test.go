package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkNetEntityCreates(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, "c1")
	require.NoError(b, err)

	b.Run("createNetworkEntities", func(b *testing.B) {
		// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
		db := pgtest.ForT(b)
		defer db.Teardown(b)

		store := postgres.New(db.DB)

		treeMgr := networktree.Singleton()
		defer treeMgr.DeleteNetworkTree(ctx, "c1")

		ds := NewEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), treeMgr, connection.ManagerSingleton())

		for _, e := range entities {
			require.NoError(b, ds.CreateExternalNetworkEntity(ctx, e, true))
		}
	})
}

func BenchmarkNetEntityCreateBatch(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, "c1")
	require.NoError(b, err)

	b.Run("createNetworkEntitiesBatch", func(b *testing.B) {
		// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
		db := pgtest.ForT(b)
		defer db.Teardown(b)

		store := postgres.New(db.DB)

		ds := NewEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), connection.ManagerSingleton())

		_, err = ds.CreateExtNetworkEntitiesForCluster(ctx, "c1", entities...)
		require.NoError(b, err)
	})
}

func BenchmarkNetEntityUpdates(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
	db := pgtest.ForT(b)
	defer db.Teardown(b)

	store := postgres.New(db.DB)
	ds := NewEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), connection.ManagerSingleton())

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, "c1")
	require.NoError(b, err)

	for _, e := range entities {
		require.NoError(b, ds.CreateExternalNetworkEntity(ctx, e, true))
	}

	b.Run("updateNetworkEntities", func(b *testing.B) {
		for _, e := range entities {
			require.NoError(b, ds.UpdateExternalNetworkEntity(ctx, e, true))
		}
	})
}
