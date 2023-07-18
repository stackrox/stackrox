package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/processindicator/search"
	postgresStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkAddIndicator(b *testing.B) {
	var indicators []*storage.ProcessIndicator
	for i := 0; i < 1000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)
	indexer := postgresStore.NewIndexer(db)
	searcher := search.New(store, indexer)

	datastore, err := New(store, plopStore, searcher, nil)
	require.NoError(b, err)

	ctx := sac.WithAllAccess(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := datastore.AddProcessIndicators(ctx, indicators...)
		require.NoError(b, err)
	}
}
