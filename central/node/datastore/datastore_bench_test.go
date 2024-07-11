package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkNodes(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))

	testDB := pgtest.ForT(b)
	nodeDS := GetTestPostgresDataStore(b, testDB.DB)
	defer testDB.Teardown(b)

	fakeNode := fixtures.GetNodeWithUniqueComponents(100, 100)
	nodes := make([]*storage.Node, 100)
	for i := 0; i < 100; i++ {
		fakeNode.Id = uuid.NewV4().String()
		fakeNode.ClusterId = fakeNode.Id
		fakeNode.ClusterName = fmt.Sprintf("c-%d", i)
		fakeNode.Name = fmt.Sprintf("node-%d", i)
		nodes[i] = fakeNode
		require.NoError(b, nodeDS.UpsertNode(ctx, fakeNode))
	}

	// Stored node is read because it contains new scan.
	b.Run("upsertNodeWithOldScan", func(b *testing.B) {
		fakeNode.Scan.ScanTime.Seconds = fakeNode.Scan.ScanTime.Seconds - 500
		for i := 0; i < b.N; i++ {
			require.NoError(b, nodeDS.UpsertNode(ctx, fakeNode))
		}
	})

	b.Run("upsertNodeWithNewScan", func(b *testing.B) {
		fakeNode.Scan.ScanTime.Seconds = fakeNode.Scan.ScanTime.Seconds + 500
		for i := 0; i < b.N; i++ {
			require.NoError(b, nodeDS.UpsertNode(ctx, fakeNode))
		}
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
