package search

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/require"
)

func BenchmarkLoad(b *testing.B) {
	tmpStore, err := bolthelper.NewTemp("alert_bench_test.db")
	require.NoError(b, err)
	s := store.New(tmpStore)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)
	idx := index.New(tmpIndex)

	searcher, err := New(s, idx)
	require.NoError(b, err)

	searchImpl := searcher.(*searcherImpl)

	for i := 0; i < 15000; i++ {
		a := fixtures.GetAlertWithID(fmt.Sprintf("%d", i))
		require.NoError(b, s.AddAlert(a))
	}

	log.Infof("Successfully loaded the DB")

	b.ResetTimer()
	// Load the store with 15k alerts and then try to build index
	for i := 0; i < b.N; i++ {
		require.NoError(b, searchImpl.buildIndex())
	}
}
