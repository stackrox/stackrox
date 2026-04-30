//go:build sql_integration

package datastore

import (
	"context"
	"slices"
	"testing"

	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
)

// addIndicators inserts all unique process indicators corresponding to the PLOPs.
func addIndicators(b *testing.B, ctx context.Context, ds DataStore, plops []*storage.ProcessListeningOnPortFromSensor) {
	indicatorMap := make(map[string]struct{})
	var initialIndicators []*storage.ProcessIndicator

	for _, plop := range plops {
		indicatorKey := id.GetIndicatorIDFromProcessIndicatorUniqueKey(plop.GetProcess())
		if _, ok := indicatorMap[indicatorKey]; !ok {
			indicatorMap[indicatorKey] = struct{}{}
			initialIndicators = append(initialIndicators, &storage.ProcessIndicator{
				Id:            indicatorKey,
				DeploymentId:  plop.GetDeploymentId(),
				PodUid:        plop.GetPodUid(),
				ClusterId:     plop.GetClusterId(),
				Namespace:     plop.GetNamespace(),
				ContainerName: plop.GetProcess().GetContainerName(),
				PodId:         plop.GetProcess().GetPodId(),
				Signal: &storage.ProcessSignal{
					Name:         plop.GetProcess().GetProcessName(),
					Args:         plop.GetProcess().GetProcessArgs(),
					ExecFilePath: plop.GetProcess().GetProcessExecFilePath(),
				},
			})
		}
	}

	indicatorBatches := slices.Chunk(initialIndicators, 1000)
	indicatorDataStore := ds.(*datastoreImpl).indicatorDataStore

	for batch := range indicatorBatches {
		if err := indicatorDataStore.AddProcessIndicators(ctx, batch...); err != nil {
			b.Fatalf("Failed to add indicator batch: %v", err)
		}
	}
}

// setupBenchmark sets up the necessary datastore and prerequisites for the benchmark.
func setupBenchmark(b *testing.B) (context.Context, DataStore, *pgtest.TestPostgres) {
	ctx := sac.WithAllAccess(sac.WithAllAccess(context.Background()))

	postgres := pgtest.ForT(b)

	store := postgresStore.NewFullStore(postgres.DB)
	indicatorStorage := processIndicatorStorage.New(postgres.DB)
	indicatorDataStore := processIndicatorDataStore.New(postgres.DB, indicatorStorage, store, nil)
	ds := New(store, indicatorDataStore, postgres)

	// Ensure required deployment exists
	deploymentDS, err := deploymentStore.GetTestPostgresDataStore(b, postgres.DB)
	if err != nil {
		b.Fatal(err)
	}
	if err := deploymentDS.UpsertDeployment(ctx, &storage.Deployment{
		Id:        fixtureconsts.Deployment1,
		Namespace: fixtureconsts.Namespace1,
		ClusterId: fixtureconsts.Cluster1,
	}); err != nil {
		require.NoError(b, err)
	}

	return ctx, ds, postgres
}

// BenchmarkAddPLOPs measures the performance of adding PLOPs to the database
func BenchmarkAddPLOPs(b *testing.B) {
	b.Run("2K PLOPs", benchmarkAddPLOPs(b, 10, 10, 10))
	b.Run("16K PLOPs", benchmarkAddPLOPs(b, 20, 20, 20))
	b.Run("250K PLOPs", benchmarkAddPLOPs(b, 50, 50, 50))
}

func benchmarkAddPLOPs(b *testing.B, nPort int, nProcess int, nPod int) func(*testing.B) {
	return func(b *testing.B) {
		// Generate the data once before the loop
		plopObjects := makeRandomPlops(nPort, nProcess, nPod, fixtureconsts.Deployment1)

		// Setup database once before the benchmark loop
		ctx, ds, postgres := setupBenchmark(b)

		// Insert all indicators once before timing - they are relatively stable
		addIndicators(b, ctx, ds, plopObjects)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Start timing for the operation we're benchmarking
			b.StartTimer()
			if err := ds.AddProcessListeningOnPort(ctx, fixtureconsts.Cluster1, plopObjects...); err != nil {
				require.NoError(b, err)
			}
			b.StopTimer()

			// Clean up PLOPs for next iteration using TRUNCATE (faster than deleting by pod UID)
			// This keeps process_indicators intact since they're added once before the benchmark
			_, err := postgres.DB.Exec(ctx, "TRUNCATE listening_endpoints")
			require.NoError(b, err)
		}
	}
}

// BenchmarkRemovePlopsByPod measures the performance of removing PLOPs by pod UID
func BenchmarkRemovePlopsByPod(b *testing.B) {
	b.Run("2K PLOPs/30 Pods", benchmarkRemovePlopsByPod(b, 10, 10, 30))
	b.Run("16K PLOPs/100 Pods", benchmarkRemovePlopsByPod(b, 20, 20, 100))
}

func benchmarkRemovePlopsByPod(b *testing.B, nPort int, nProcess int, nPod int) func(*testing.B) {
	return func(b *testing.B) {
		// Setup database once before the benchmark loop
		ctx, ds, postgres := setupBenchmark(b)

		// Generate a dataset with nPod pods (background data)
		allPodData := makeRandomPlops(nPort, nProcess, nPod, fixtureconsts.Deployment1)

		// Generate data for one additional pod that we'll repeatedly insert/delete
		targetPodData := makeRandomPlops(nPort, nProcess, 1, fixtureconsts.Deployment1)
		targetPodUID := targetPodData[0].GetPodUid()

		// Insert indicators for all data once
		addIndicators(b, ctx, ds, append(allPodData, targetPodData...))

		// Insert the background data once (stays for all iterations)
		if err := ds.AddProcessListeningOnPort(ctx, fixtureconsts.Cluster1, allPodData...); err != nil {
			require.NoError(b, err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		// Benchmark: insert target pod's PLOPs then delete them
		for i := 0; i < b.N; i++ {
			// Insert target pod's PLOPs (not timed)
			b.StopTimer()
			if err := ds.AddProcessListeningOnPort(ctx, fixtureconsts.Cluster1, targetPodData...); err != nil {
				require.NoError(b, err)
			}

			_, err := postgres.DB.Exec(ctx, "ANALYZE listening_endpoints")
			require.NoError(b, err)

			b.StartTimer()

			// Benchmark the deletion
			if err := ds.RemovePlopsByPod(ctx, targetPodUID); err != nil {
				require.NoError(b, err)
			}
		}
	}
}
