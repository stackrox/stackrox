package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/networkgraph/config/datastore/mocks"
	store "github.com/stackrox/stackrox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/generated/storage"
	pkgNet "github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/networkgraph/testutils"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkNetEntityCreates(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, "c1")
	require.NoError(b, err)

	b.Run("createNetworkEntities", func(b *testing.B) {
		// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
		db, err := rocksdb.NewTemp(b.Name())
		require.NoError(b, err)
		defer rocksdbtest.TearDownRocksDB(db)

		store, err := store.New(db)
		require.NoError(b, err)

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
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, "c1")
	require.NoError(b, err)

	b.Run("createNetworkEntitiesBatch", func(b *testing.B) {

		db, err := rocksdb.NewTemp(b.Name())
		require.NoError(b, err)
		defer rocksdbtest.TearDownRocksDB(db)

		store, err := store.New(db)
		require.NoError(b, err)

		ds := NewEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), connection.ManagerSingleton())

		_, err = ds.CreateExtNetworkEntitiesForCluster(ctx, "c1", entities...)
		require.NoError(b, err)
	})
}

func BenchmarkNetEntityUpdates(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	db, err := rocksdb.NewTemp(b.Name())
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	store, err := store.New(db)
	require.NoError(b, err)
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
