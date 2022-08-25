package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	rocksDBStore "github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func BenchmarkDBs(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	db, err := rocksdb.NewTemp("alert_bench_test")
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)
	defer tmpIndex.Close()

	s := rocksDBStore.New(db)
	idx := index.New(tmpIndex)
	datastore, err := New(s, idx, search.New(s, idx))
	require.NoError(b, err)
	datastoreImpl := datastore.(*datastoreImpl)

	var ids []string
	for i := 0; i < 15000; i++ {
		id := fmt.Sprintf("%d", i)
		ids = append(ids, id)
		a := fixtures.GetAlertWithID(id)
		require.NoError(b, s.Upsert(ctx, a))
	}

	log.Info("Successfully loaded the DB")

	b.Run("rocksdb", func(b *testing.B) {
		// Load the store with 15k alerts and then try to build index
		for i := 0; i < b.N; i++ {
			require.NoError(b, datastoreImpl.buildIndex(ctx))
		}
	})

	b.Run("markStale", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, id := range ids {
				require.NoError(b, datastore.MarkAlertStale(ctx, id))
			}
		}
	})

	b.Run("markStaleBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := datastore.MarkAlertStaleBatch(ctx, ids...)
			require.NoError(b, err)
		}
	})
}
