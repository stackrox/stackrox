//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	postgresStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	processBaselineResultsDataStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

func BenchmarkAddProcessBaselines(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
		),
	)

	pgtestbase := pgtest.ForT(b)
	pool := pgtestbase.DB
	storage := postgresStore.New(pool)

	baselineResultsStore := processBaselineResultsDataStore.GetTestPostgresDataStore(b, pool)
	indicatorStore := processIndicatorDataStore.GetTestPostgresDataStore(b, pool)
	datastore := New(storage, baselineResultsStore, indicatorStore)

	b.Run("Add process baselines 10", benchmarkAddProcessBaselines(ctx, datastore, 10))
	b.Run("Add process baselines 100", benchmarkAddProcessBaselines(ctx, datastore, 100))
	b.Run("Add process baselines 1000", benchmarkAddProcessBaselines(ctx, datastore, 1000))
	b.Run("Add process baselines 10000", benchmarkAddProcessBaselines(ctx, datastore, 10000))
}

func createRandomBaseline() *storage.ProcessBaseline {

	return &storage.ProcessBaseline{
		Id: uuid.NewV4().String(),
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  uuid.NewV4().String(),
			ContainerName: uuid.NewV4().String(),
			ClusterId:     uuid.NewV4().String(),
			Namespace:     uuid.NewV4().String(),
		},
	}
}

func createRandomBaselines(numBaselines int) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, numBaselines)

	for i := range numBaselines {
		baselines[i] = createRandomBaseline()
	}

	return baselines
}

func getIds(baselines []*storage.ProcessBaseline) []string {
	numBaselines := len(baselines)
	ids := make([]string, numBaselines)

	for i := range numBaselines {
		ids[i] = baselines[i].Id
	}

	return ids
}

func benchmarkAddProcessBaselines(ctx context.Context, datastore DataStore, numBaselines int) func(*testing.B) {
	baselines := createRandomBaselines(numBaselines)
	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, baseline := range baselines {
				datastore.AddProcessBaseline(ctx, baseline)
			}
			b.StopTimer()
			ids := getIds(baselines)
			datastore.RemoveProcessBaselinesByIDs(ctx, ids)
			b.StartTimer()
		}
	}
}
