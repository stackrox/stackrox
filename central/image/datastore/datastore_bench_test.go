package datastore

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkImages(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
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

	imageDS := New(dacky, concurrency.NewKeyFence(), bleveIndex, bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())

	// Generate CVEs and components for the image.
	var components []*storage.EmbeddedImageScanComponent
	for i := 0; i < 100; i++ {
		var vulns []*storage.EmbeddedVulnerability
		for j := 0; j < 1000; j++ {
			vuln := &storage.EmbeddedVulnerability{
				Cve: strconv.Itoa(i) + strconv.Itoa(j),
			}
			vulns = append(vulns, vuln)
		}

		component := &storage.EmbeddedImageScanComponent{
			Name:    strconv.Itoa(i),
			Version: strconv.Itoa(i),
			Vulns:   vulns,
		}
		components = append(components, component)
	}

	image1 := &storage.Image{
		Id: "image1",
		Scan: &storage.ImageScan{
			ScanTime:   types.TimestampNow(),
			Components: components,
		},
	}

	require.NoError(b, imageDS.UpsertImage(ctx, image1))

	// Stored image is read because it contains new scan.
	b.Run("upsertImageWithOldScan", func(b *testing.B) {
		image1.Scan.ScanTime.Seconds = image1.Scan.ScanTime.Seconds - 500
		for i := 0; i < b.N; i++ {
			err = imageDS.UpsertImage(ctx, image1)
		}
		require.NoError(b, err)
	})

	b.Run("upsertImageWithNewScan", func(b *testing.B) {
		image1.Scan.ScanTime.Seconds = image1.Scan.ScanTime.Seconds + 500
		for i := 0; i < b.N; i++ {
			err = imageDS.UpsertImage(ctx, image1)
		}
		require.NoError(b, err)
	})
}
