package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/processindicator/index"
	"github.com/stackrox/stackrox/central/processindicator/search"
	rocksdbStore "github.com/stackrox/stackrox/central/processindicator/store/rocksdb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkAddIndicator(b *testing.B) {
	var indicators []*storage.ProcessIndicator
	for i := 0; i < 1000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		indicators = append(indicators, pi)
	}

	db, err := rocksdb.NewTemp(testutils.DBFileNameForT(b))
	require.NoError(b, err)

	store := rocksdbStore.New(db)
	tmpIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)

	indexer := index.New(tmpIndex)
	searcher := search.New(store, indexer)

	datastore, err := New(store, indexer, searcher, nil)
	require.NoError(b, err)

	ctx := sac.WithAllAccess(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := datastore.AddProcessIndicators(ctx, indicators...)
		require.NoError(b, err)
	}
}
