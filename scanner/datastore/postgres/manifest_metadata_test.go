//go:build scanner_db_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ManifestMetadata_Init(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "manifest_metadata_init_test")
	ccStore, err := postgres.InitPostgresIndexerStore(ctx, pool, true)
	require.NoError(t, err)
	store, err := InitPostgresIndexerMetadataStore(ctx, pool, true, IndexerMetadataStoreOpts{IndexerStore: ccStore})
	require.NoError(t, err)

	shas := []string{
		"sha512:de1ab9379bccc4afea75ef6b5a53e1ca867e97bd2edfaa61256368a579249518a283b81d95d1bdcdebdb8d96fe7f0219daeda8941c9cbddf64b6e3c543389d14",
		"sha512:de1ab9379bccc4afea75ef6b5a53e1ca867e97bd2edfaa61256368a579249518a283b81d95d1bdcdebdb8d96fe7f0219daeda8941c9cbddf64b6e3c543389d15",
		"sha512:de1ab9379bccc4afea75ef6b5a53e1ca867e97bd2edfaa61256368a579249518a283b81d95d1bdcdebdb8d96fe7f0219daeda8941c9cbddf64b6e3c543389d16",
		"sha512:de1ab9379bccc4afea75ef6b5a53e1ca867e97bd2edfaa61256368a579249518a283b81d95d1bdcdebdb8d96fe7f0219daeda8941c9cbddf64b6e3c543389d17",
	}

	for _, sha := range shas {
		err = ccStore.PersistManifest(ctx, claircore.Manifest{
			Hash: claircore.MustParseDigest(sha),
			Layers: []*claircore.Layer{
				{
					Hash: claircore.MustParseDigest(sha),
				},
			},
		})
		require.NoError(t, err)
	}

	ms, err := store.Init(ctx)
	require.NoError(t, err)
	assert.Len(t, ms, 4)
	assert.ElementsMatch(t, shas, ms)
}

func Test_ManifestMetadata(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "manifest_metadata_test")
	store, err := InitPostgresIndexerMetadataStore(ctx, pool, true, IndexerMetadataStoreOpts{})
	require.NoError(t, err)
	now := time.Now()

	// Add a manifest which should not be deleted.
	err = store.StoreManifest(ctx, "sha512:abc", now.Add(1*time.Hour))
	assert.NoError(t, err)
	// Run GC but do not delete.
	ids, err := store.GCManifests(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 0)

	// Add a manifest to be deleted.
	// First, add it an hour ahead.
	err = store.StoreManifest(ctx, "sha512:def", now.Add(1*time.Hour))
	assert.NoError(t, err)
	// Next, add it an hour behind.
	// This ensures the row is overwritten.
	err = store.StoreManifest(ctx, "sha512:def", now.Add(-1*time.Hour))
	assert.NoError(t, err)
	// Delete the manifest.
	ids, err = store.GCManifests(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "sha512:def", ids[0])

	// Add a manifest which manages to have the same time as now.
	err = store.StoreManifest(ctx, "sha512:ghi", now)
	assert.NoError(t, err)
	// This manifest gets lucky, and it does not get deleted. Yet...
	// Note the manifest from the previous test is also not deleted (again).
	ids, err = store.GCManifests(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 0)

	// More complex test.
	// Note: according to https://www.postgresql.org/docs/15/datatype-datetime.html
	// PostgreSQL timestamps are at microsecond resolution.
	err = store.StoreManifest(ctx, "sha512:jkl", now.Add(-1*time.Microsecond))
	assert.NoError(t, err)
	err = store.StoreManifest(ctx, "sha512:mno", now.Add(1*time.Microsecond))
	assert.NoError(t, err)
	err = store.StoreManifest(ctx, "sha512:pqr", now.Add(2*time.Microsecond))
	assert.NoError(t, err)
	err = store.StoreManifest(ctx, "sha512:stu", time.Time{})
	assert.NoError(t, err)
	err = store.StoreManifest(ctx, "sha512:vwx", time.Time{}.Add(-1*time.Microsecond))
	assert.NoError(t, err)
	ids, err = store.GCManifests(ctx, now)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.ElementsMatch(t, []string{"sha512:jkl", "sha512:stu", "sha512:vwx"}, ids)

	// Delete everything after 5 years (24 hours/day * 365 days/year * 5 years = 43,800 hours)
	ids, err = store.GCManifests(ctx, now.Add(43_800*time.Hour))
	require.NoError(t, err)
	// There should have been 4 remaining rows.
	assert.Len(t, ids, 4)
	assert.ElementsMatch(t, []string{"sha512:abc", "sha512:ghi", "sha512:mno", "sha512:pqr"}, ids)
}
