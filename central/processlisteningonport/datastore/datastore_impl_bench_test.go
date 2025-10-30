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

func getUniquePodsFromPlops(plops []*storage.ProcessListeningOnPortFromSensor) []string {
	pods := make([]string, 0, len(plops))
	podSet := make(map[string]struct{})

	for _, plop := range plops {
		if _, exists := podSet[plop.GetPodUid()]; !exists {
			pods = append(pods, plop.GetPodUid())
			podSet[plop.GetPodUid()] = struct{}{}
		}
	}

	return pods
}

func removeAllPlops(ctx context.Context, ds DataStore, plops []*storage.ProcessListeningOnPortFromSensor) error {
	pods := getUniquePodsFromPlops(plops)

	for _, pod := range pods {
		err := ds.RemovePlopsByPod(ctx, pod)
		if err != nil {
			return err
		}
	}

	return nil
}

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
	b.Logf("Inserted %d unique Process Indicators.", len(initialIndicators))
}

// setupBenchmark sets up the necessary datastore and prerequisites for the benchmark.
func setupBenchmark(b *testing.B) (context.Context, DataStore) {
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

	return ctx, ds
}

// BenchmarkAddPLOPs measures the performance of adding PLOPs to the database
func BenchmarkAddPLOPs(b *testing.B) {
	b.Run("2K PLOPs", benchmarkAddPLOPs(b, 10, 10, 10))
	b.Run("16K PLOPs", benchmarkAddPLOPs(b, 20, 20, 20))
}

func benchmarkAddPLOPs(b *testing.B, nPort int, nProcess int, nPod int) func(*testing.B) {
	return func(b *testing.B) {
		// Generate the data once before the loop
		plopObjects := makeRandomPlops(nPort, nProcess, nPod, fixtureconsts.Deployment1)
		b.Logf("Benchmarking processing of %d new PLOP objects...", len(plopObjects))

		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx, ds := setupBenchmark(b)
			// Insert all indicators so lookups hit existing data
			addIndicators(b, ctx, ds, plopObjects)

			b.StartTimer()
			if err := ds.AddProcessListeningOnPort(ctx, fixtureconsts.Cluster1, plopObjects...); err != nil {
				require.NoError(b, err)
			}
			b.StopTimer()
		}
	}
}
