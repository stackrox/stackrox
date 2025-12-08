package datastore

import (
	"context"
	"math/rand"
	"testing"

	postgresStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := datastore.AddProcessIndicators(ctx, indicators...)
		require.NoError(b, err)
	}
}

func BenchmarkProcessIndicators(b *testing.B) {
	// Create pod IDs
	podID1 := uuid.NewV4().String()
	podID2 := uuid.NewV4().String()
	podID3 := uuid.NewV4().String()

	// Create indicators with both deployment (55/25/20) and pod distribution (55/25/20)
	// This creates a matrix where we have all combinations
	var allIndicators []*storage.ProcessIndicator

	// Deployment1: 5,500 total
	//   PodID1: 3,025 (55%)
	//   PodID2: 1,375 (25%)
	//   PodID3: 1,100 (20%)
	for i := 0; i < 5500; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.DeploymentId = fixtureconsts.Deployment1
		if i < 3025 {
			pi.PodUid = podID1
		} else if i < 4400 {
			pi.PodUid = podID2
		} else {
			pi.PodUid = podID3
		}
		allIndicators = append(allIndicators, pi)
	}

	// Deployment2: 2,500 total
	//   PodID1: 1,375 (55%)
	//   PodID2: 625 (25%)
	//   PodID3: 500 (20%)
	for i := 0; i < 2500; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.DeploymentId = fixtureconsts.Deployment2
		if i < 1375 {
			pi.PodUid = podID1
		} else if i < 2000 {
			pi.PodUid = podID2
		} else {
			pi.PodUid = podID3
		}
		allIndicators = append(allIndicators, pi)
	}

	// Deployment3: 2,000 total
	//   PodID1: 1,100 (55%)
	//   PodID2: 500 (25%)
	//   PodID3: 400 (20%)
	for i := 0; i < 2000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.DeploymentId = fixtureconsts.Deployment3
		if i < 1100 {
			pi.PodUid = podID1
		} else if i < 1600 {
			pi.PodUid = podID2
		} else {
			pi.PodUid = podID3
		}
		allIndicators = append(allIndicators, pi)
	}

	// Shuffle to randomize the distribution
	rand.Shuffle(len(allIndicators), func(i, j int) {
		allIndicators[i], allIndicators[j] = allIndicators[j], allIndicators[i]
	})

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add all data once
	err := datastore.AddProcessIndicators(ctx, allIndicators...)
	require.NoError(b, err)

	// ==================== SEARCH PHASE ====================
	// Search benchmarks - non-destructive, can run multiple times
	b.Run("Search/ByDeployment1", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment1).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Search/ByDeployment2", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment2).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Search/ByDeployment3", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment3).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Search/ByPodID1", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID1).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Search/ByPodID2", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID2).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Search/ByPodID3", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID3).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	// ==================== DELETE PHASE ====================
	// Simple delete benchmarks - measure the actual delete operation without re-add overhead
	// Collect data to delete after searches are complete
	d1Query := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, fixtureconsts.Deployment1).
		ProtoQuery()
	d1Results, err := datastore.SearchRawProcessIndicators(ctx, d1Query)
	require.NoError(b, err)
	require.True(b, len(d1Results) > 0)

	d1DeleteIDs := make([]string, len(d1Results))
	for i, r := range d1Results {
		d1DeleteIDs[i] = r.Id
	}

	p2Query := search.NewQueryBuilder().
		AddExactMatches(search.PodUID, podID2).
		ProtoQuery()
	p2Results, err := datastore.SearchRawProcessIndicators(ctx, p2Query)
	require.NoError(b, err)
	require.True(b, len(p2Results) > 0)

	p2DeleteIDs := make([]string, len(p2Results))
	for i, r := range p2Results {
		p2DeleteIDs[i] = r.Id
	}

	b.Run("Delete/ByDeployment1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicators(ctx, d1DeleteIDs)
			require.NoError(b, err)
		}
	})

	b.Run("Delete/ByPodID2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicators(ctx, p2DeleteIDs)
			require.NoError(b, err)
		}
	})
}

func makeStringSlice(s string, count int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = s
	}
	return result
}
