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
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	globalAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph),
		))
)

func BenchmarkNetEntityCreates(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
	db := pgtest.ForT(b)

	store := postgres.New(db.DB)

	treeMgr := networktree.Singleton()
	defer treeMgr.DeleteNetworkTree(globalAccessCtx, testconsts.Cluster1)

	dataPusher := newNetworkEntityPusher(connection.ManagerSingleton())

	ds := newEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), treeMgr, dataPusher)

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, b.N, testconsts.Cluster1)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		require.NoError(b, ds.CreateExternalNetworkEntity(globalAccessCtx, entities[i], true))
	}
}

func BenchmarkNetEntityCreateBatch(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
	db := pgtest.ForT(b)

	store := postgres.New(db.DB)
	dataPusher := newNetworkEntityPusher(connection.ManagerSingleton())

	ds := newEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), dataPusher)

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, 10000, testconsts.Cluster1)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = ds.CreateExtNetworkEntitiesForCluster(globalAccessCtx, testconsts.Cluster1, entities...)
		require.NoError(b, err)

		b.StopTimer()
		require.NoError(b, ds.DeleteExternalNetworkEntitiesForCluster(globalAccessCtx, testconsts.Cluster1))
		b.StartTimer()
	}
}

func BenchmarkNetEntityUpdates(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()

	// Need to recreate the DB to avoid failure due to key conflicts from the reruns.
	db := pgtest.ForT(b)

	store := postgres.New(db.DB)
	dataPusher := newNetworkEntityPusher(connection.ManagerSingleton())
	ds := newEntityDataStore(store, mocks.NewMockDataStore(mockCtrl), networktree.Singleton(), dataPusher)

	entities, err := testutils.GenRandomExtSrcNetworkEntity(pkgNet.IPv4, b.N, testconsts.Cluster1)
	require.NoError(b, err)

	for _, e := range entities {
		require.NoError(b, ds.CreateExternalNetworkEntity(globalAccessCtx, e, true))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		require.NoError(b, ds.UpdateExternalNetworkEntity(globalAccessCtx, entities[i], true))
	}
}
