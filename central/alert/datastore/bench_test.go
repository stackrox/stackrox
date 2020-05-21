package datastore

import (
	"fmt"
	"testing"

	commentsStore "github.com/stackrox/rox/central/alert/datastore/internal/commentsstore"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	badgerStore "github.com/stackrox/rox/central/alert/datastore/internal/store/badger"
	rocksDBStore "github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/require"
)

func BenchmarkDBs(b *testing.B) {
	b.Run("badger", func(b *testing.B) {
		db, _, err := badgerhelper.NewTemp("alert_bench_test")
		require.NoError(b, err)
		benchmarkLoad(b, badgerStore.New(db), nil)
	})

	b.Run("rocksdb", func(b *testing.B) {
		db, _, err := rocksdb.NewTemp("alert_bench_test")
		require.NoError(b, err)
		benchmarkLoad(b, rocksDBStore.NewFullStore(db), nil)
	})
}

func benchmarkLoad(b *testing.B, s store.Store, c commentsStore.Store) {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)
	idx := index.New(tmpIndex)

	datastore, err := New(s, c, idx, search.New(s, idx))
	require.NoError(b, err)
	datastoreImpl := datastore.(*datastoreImpl)

	for i := 0; i < 15000; i++ {
		a := fixtures.GetAlertWithID(fmt.Sprintf("%d", i))
		require.NoError(b, s.Upsert(a))
	}

	log.Info("Successfully loaded the DB")

	b.ResetTimer()
	// Load the store with 15k alerts and then try to build index
	for i := 0; i < b.N; i++ {
		require.NoError(b, datastoreImpl.buildIndex())
	}
}
