package datastore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkNodes(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))

	var err error
	var nodeDS DataStore
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testDB := pgtest.ForT(b)
		nodeDS, err = GetTestPostgresDataStore(b, testDB.DB)
		require.NoError(b, err)
		defer testDB.Teardown(b)
	} else {
		db, err := rocksdb.NewTemp(b.Name())
		require.NoError(b, err)
		defer rocksdbtest.TearDownRocksDB(db)

		dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
		require.NoError(b, err)

		tempPath := b.TempDir()
		blevePath := filepath.Join(tempPath, "scorch.bleve")
		bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
		require.NoError(b, err)

		nodeDS, err = GetTestRocksBleveDataStore(b, db, bleveIndex, dacky, concurrency.NewKeyFence())
		require.NoError(b, err)
	}

	fakeNode := fixtures.GetNodeWithUniqueComponents(100, 100)
	nodes := make([]*storage.Node, 100)
	for i := 0; i < 100; i++ {
		fakeNode.Id = uuid.NewV4().String()
		fakeNode.ClusterId = fakeNode.Id
		fakeNode.ClusterName = fmt.Sprintf("c-%d", i)
		fakeNode.Name = fmt.Sprintf("node-%d", i)
		nodes[i] = fakeNode
		require.NoError(b, nodeDS.UpsertNode(ctx, fakeNode, false))
	}

	// Stored node is read because it contains new scan.
	b.Run("upsertNodeWithOldScan", func(b *testing.B) {
		fakeNode.Scan.ScanTime.Seconds = fakeNode.Scan.ScanTime.Seconds - 500
		for i := 0; i < b.N; i++ {
			err = nodeDS.UpsertNode(ctx, fakeNode, false)
		}
		require.NoError(b, err)
	})

	b.Run("upsertNodeWithNewScan", func(b *testing.B) {
		fakeNode.Scan.ScanTime.Seconds = fakeNode.Scan.ScanTime.Seconds + 500
		for i := 0; i < b.N; i++ {
			err = nodeDS.UpsertNode(ctx, fakeNode, false)
		}
		require.NoError(b, err)
	})

	b.Run("searchAll", func(b *testing.B) {
		results, err := nodeDS.SearchRawNodes(ctx, search.EmptyQuery())
		require.NoError(b, err)
		require.NotNil(b, results)
	})

	b.Run("searchForCluster", func(b *testing.B) {
		results, err := nodeDS.SearchRawNodes(ctx, search.NewQueryBuilder().AddExactMatches(search.ClusterID, nodes[0].ClusterId).ProtoQuery())
		require.NoError(b, err)
		require.NotNil(b, results)
	})

	b.Run("deleteForClusters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			idx := i % len(nodes)
			err := nodeDS.DeleteAllNodesForCluster(ctx, nodes[idx].ClusterId)
			require.NoError(b, err)
		}
	})
}
