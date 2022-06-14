package datastore

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/ranking"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkNodes(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))

	db, err := rocksdb.NewTemp(b.Name())
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(b, err)

	tempPath := b.TempDir()
	blevePath := filepath.Join(tempPath, "scorch.bleve")
	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
	require.NoError(b, err)

	nodeDS := New(dacky, concurrency.NewKeyFence(), bleveIndex, nil, ranking.NewRanker(), ranking.NewRanker())

	// Generate CVEs and components for the node.
	components := make([]*storage.EmbeddedNodeScanComponent, 0, 1000)
	for i := 0; i < 100; i++ {
		vulns := make([]*storage.EmbeddedVulnerability, 0, 1000)
		for j := 0; j < 1000; j++ {
			vuln := &storage.EmbeddedVulnerability{
				Cve: strconv.Itoa(i) + strconv.Itoa(j),
			}
			vulns = append(vulns, vuln)
		}

		component := &storage.EmbeddedNodeScanComponent{
			Name:    strconv.Itoa(i),
			Version: strconv.Itoa(i),
			Vulns:   vulns,
		}
		components = append(components, component)
	}

	node1 := &storage.Node{
		Id: "node1",
		Scan: &storage.NodeScan{
			ScanTime:   types.TimestampNow(),
			Components: components,
		},
	}

	require.NoError(b, nodeDS.UpsertNode(ctx, node1))

	// Stored node is read because it contains new scan.
	b.Run("upsertNodeWithOldScan", func(b *testing.B) {
		node1.Scan.ScanTime.Seconds = node1.Scan.ScanTime.Seconds - 500
		for i := 0; i < b.N; i++ {
			err = nodeDS.UpsertNode(ctx, node1)
		}
		require.NoError(b, err)
	})

	b.Run("upsertNodeWithNewScan", func(b *testing.B) {
		node1.Scan.ScanTime.Seconds = node1.Scan.ScanTime.Seconds + 500
		for i := 0; i < b.N; i++ {
			err = nodeDS.UpsertNode(ctx, node1)
		}
		require.NoError(b, err)
	})
}
