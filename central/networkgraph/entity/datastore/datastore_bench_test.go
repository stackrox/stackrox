package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	store "github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
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
		// Need to recreate the DB to avoid failure to key conflicts from the rerun.
		db, err := rocksdb.NewTemp(b.Name())
		require.NoError(b, err)
		defer rocksdbtest.TearDownRocksDB(db)

		store, err := store.New(db)
		require.NoError(b, err)
		ds := NewEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), connection.ManagerSingleton())

		for _, e := range entities {
			require.NoError(b, ds.CreateExternalNetworkEntity(ctx, e, true))
		}
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
