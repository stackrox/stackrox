package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/processindicator/search"
	postgresStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkAddIndicator(b *testing.B) {
	var indicators []*storage.ProcessIndicator
	for i := 0; i < 100000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)
	searcher := search.New(store)

	datastore := New(store, plopStore, searcher, nil)

	ctx := sac.WithAllAccess(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := datastore.AddProcessIndicators(ctx, indicators...)
		require.NoError(b, err)
	}
}

func BenchmarkSearchIndicator(b *testing.B) {
	var indicators []*storage.ProcessIndicator
	for i := 0; i < 10000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		// spreading these across some deployments to set up search test
		switch i % 3 {
		case 0:
			pi.DeploymentId = fixtureconsts.Deployment1
		case 1:
			pi.DeploymentId = fixtureconsts.Deployment2
		case 2:
			pi.DeploymentId = fixtureconsts.Deployment3
		}

		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)
	searcher := search.New(store)

	datastore := New(store, plopStore, searcher, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add the data first.
	err := datastore.AddProcessIndicators(ctx, indicators...)
	require.NoError(b, err)

	b.ResetTimer()
	query := pgSearch.NewQueryBuilder().AddExactMatches(pgSearch.DeploymentID, fixtureconsts.Deployment1).ProtoQuery()
	for i := 0; i < b.N; i++ {
		results, err := datastore.SearchRawProcessIndicators(ctx, query)
		require.NoError(b, err)
		require.True(b, len(results) > 0)
	}
}
