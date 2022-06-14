package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	rocksDBStore "github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkDBs(b *testing.B) {
	b.Run("rocksdb", func(b *testing.B) {
		db, err := rocksdb.NewTemp("alert_bench_test")
		defer rocksdbtest.TearDownRocksDB(db)

		require.NoError(b, err)
		benchmarkLoad(b, rocksDBStore.New(db))
	})
}

func benchmarkLoad(b *testing.B, s store.Store) {
	ctx := context.TODO()
	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)
	idx := index.New(tmpIndex)

	datastore, err := New(s, idx, search.New(s, idx))
	require.NoError(b, err)
	datastoreImpl := datastore.(*datastoreImpl)

	for i := 0; i < 15000; i++ {
		a := fixtures.GetAlertWithID(fmt.Sprintf("%d", i))
		require.NoError(b, s.Upsert(ctx, a))
	}

	log.Info("Successfully loaded the DB")

	b.ResetTimer()
	// Load the store with 15k alerts and then try to build index
	for i := 0; i < b.N; i++ {
		require.NoError(b, datastoreImpl.buildIndex(ctx))
	}
}
