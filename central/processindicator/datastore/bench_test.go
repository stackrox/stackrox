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

func BenchmarkSearchIndicator(b *testing.B) {
	// Create a randomized distribution of deployments (55/25/20)
	deployments := make([]string, 0, 10000)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment1, 5500)...)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment2, 2500)...)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment3, 2000)...)

	// Shuffle to randomize the distribution
	rand.Shuffle(len(deployments), func(i, j int) {
		deployments[i], deployments[j] = deployments[j], deployments[i]
	})

	var indicators []*storage.ProcessIndicator
	for i := 0; i < 10000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.DeploymentId = deployments[i]
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add the data first.
	err := datastore.AddProcessIndicators(ctx, indicators...)
	require.NoError(b, err)

	b.Run("Deployment1", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment1).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Deployment2", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment2).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("Deployment3", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, fixtureconsts.Deployment3).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})
}

func BenchmarkSearchIndicatorByPodID(b *testing.B) {
	podID1 := uuid.NewV4().String()
	podID2 := uuid.NewV4().String()
	podID3 := uuid.NewV4().String()

	// Create a randomized distribution of pod IDs (55/25/20)
	podIDs := make([]string, 0, 10000)
	podIDs = append(podIDs, makeStringSlice(podID1, 5500)...)
	podIDs = append(podIDs, makeStringSlice(podID2, 2500)...)
	podIDs = append(podIDs, makeStringSlice(podID3, 2000)...)

	// Shuffle to randomize the distribution
	rand.Shuffle(len(podIDs), func(i, j int) {
		podIDs[i], podIDs[j] = podIDs[j], podIDs[i]
	})

	var indicators []*storage.ProcessIndicator
	for i := 0; i < 10000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.PodUid = podIDs[i]
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add the data first.
	err := datastore.AddProcessIndicators(ctx, indicators...)
	require.NoError(b, err)

	b.Run("PodID1", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID1).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("PodID2", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID2).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})

	b.Run("PodID3", func(b *testing.B) {
		query := search.NewQueryBuilder().AddExactMatches(search.PodUID, podID3).ProtoQuery()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawProcessIndicators(ctx, query)
			require.NoError(b, err)
			require.True(b, len(results) > 0)
		}
	})
}

func BenchmarkDeleteIndicatorByDeployment(b *testing.B) {
	// Create a randomized distribution of deployments (55/25/20)
	deployments := make([]string, 0, 10000)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment1, 5500)...)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment2, 2500)...)
	deployments = append(deployments, makeStringSlice(fixtureconsts.Deployment3, 2000)...)

	// Shuffle to randomize the distribution
	rand.Shuffle(len(deployments), func(i, j int) {
		deployments[i], deployments[j] = deployments[j], deployments[i]
	})

	var indicators []*storage.ProcessIndicator
	for i := 0; i < 10000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.DeploymentId = deployments[i]
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add the data first.
	err := datastore.AddProcessIndicators(ctx, indicators...)
	require.NoError(b, err)

	// Group indicators by deployment for deletion
	deployment1Indicators := make([]string, 0, 5500)
	deployment2Indicators := make([]string, 0, 2500)
	deployment3Indicators := make([]string, 0, 2000)

	for _, indicator := range indicators {
		switch indicator.DeploymentId {
		case fixtureconsts.Deployment1:
			deployment1Indicators = append(deployment1Indicators, indicator.Id)
		case fixtureconsts.Deployment2:
			deployment2Indicators = append(deployment2Indicators, indicator.Id)
		case fixtureconsts.Deployment3:
			deployment3Indicators = append(deployment3Indicators, indicator.Id)
		}
	}

	b.Run("Deployment1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicators(ctx, deployment1Indicators)
			require.NoError(b, err)
		}
	})

	b.Run("Deployment2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicators(ctx, deployment2Indicators)
			require.NoError(b, err)
		}
	})

	b.Run("Deployment3", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicators(ctx, deployment3Indicators)
			require.NoError(b, err)
		}
	})
}

func BenchmarkDeleteIndicatorByPodID(b *testing.B) {
	podID1 := uuid.NewV4().String()
	podID2 := uuid.NewV4().String()
	podID3 := uuid.NewV4().String()

	// Create a randomized distribution of pod IDs (55/25/20)
	podIDs := make([]string, 0, 10000)
	podIDs = append(podIDs, makeStringSlice(podID1, 5500)...)
	podIDs = append(podIDs, makeStringSlice(podID2, 2500)...)
	podIDs = append(podIDs, makeStringSlice(podID3, 2000)...)

	// Shuffle to randomize the distribution
	rand.Shuffle(len(podIDs), func(i, j int) {
		podIDs[i], podIDs[j] = podIDs[j], podIDs[i]
	})

	var indicators []*storage.ProcessIndicator
	for i := 0; i < 10000; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.Id = uuid.NewV4().String()
		pi.PodUid = podIDs[i]
		indicators = append(indicators, pi)
	}

	db := pgtest.ForT(b)
	store := postgresStore.New(db)
	plopStore := plopStore.New(db)

	datastore := New(db, store, plopStore, nil)

	ctx := sac.WithAllAccess(context.Background())
	// Add the data first.
	err := datastore.AddProcessIndicators(ctx, indicators...)
	require.NoError(b, err)

	b.Run("PodID1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicatorsByPod(ctx, podID1)
			require.NoError(b, err)
		}
	})

	b.Run("PodID2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicatorsByPod(ctx, podID2)
			require.NoError(b, err)
		}
	})

	b.Run("PodID3", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := datastore.RemoveProcessIndicatorsByPod(ctx, podID3)
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
