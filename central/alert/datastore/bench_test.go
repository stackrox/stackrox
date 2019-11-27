package datastore

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	badgerStore "github.com/stackrox/rox/central/alert/datastore/internal/store/badger"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/require"
)

func BenchmarkDBs(b *testing.B) {
	b.Run("badger", func(b *testing.B) {
		db, _, err := badgerhelper.NewTemp("alert_bench_test")
		require.NoError(b, err)
		benchmarkLoad(b, badgerStore.New(db))
	})
}

func benchmarkLoad(b *testing.B, s store.Store) {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)
	idx := index.New(tmpIndex)

	datastore, err := New(s, idx, search.New(s, idx))
	require.NoError(b, err)
	datastoreImpl := datastore.(*datastoreImpl)

	for i := 0; i < 15000; i++ {
		a := fixtures.GetAlertWithID(fmt.Sprintf("%d", i))
		require.NoError(b, s.UpsertAlert(a))
	}

	log.Info("Successfully loaded the DB")

	b.ResetTimer()
	// Load the store with 15k alerts and then try to build index
	for i := 0; i < b.N; i++ {
		require.NoError(b, datastoreImpl.buildIndex())
	}
}
