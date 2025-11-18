package lifecycle

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

const (
	// Environment variable to specify a pre-seeded database name
	preSeededDBEnvVar = "BENCHMARK_PRESEEDED_DB"
)

// BenchmarkConfig holds configuration for benchmark tests
type BenchmarkConfig struct {
	NumDeployments       int
	NumPodsPerDeployment int
	NumProcessesPerPod   int
}

// seedBenchmarkData populates the database with test data
func seedBenchmarkData(
	ctx context.Context,
	t *testing.T,
	deploymentDS deploymentDatastore.DataStore,
	processDS processIndicatorDatastore.DataStore,
	config BenchmarkConfig,
) {
	t.Helper()

	// Use all-access context for seeding
	ctx = sac.WithAllAccess(ctx)

	clusterID := uuid.NewV4().String()

	// Create deployments, pods, and process indicators
	for i := 0; i < config.NumDeployments; i++ {
		deploymentID := uuid.NewV4().String()
		deployment := &storage.Deployment{
			Id:        deploymentID,
			Name:      fmt.Sprintf("deployment-%d", i),
			ClusterId: clusterID,
			Namespace: fmt.Sprintf("namespace-%d", i%10),
			Containers: []*storage.Container{
				{
					Name:  "main-container",
					Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "nginx:latest"}},
				},
			},
		}

		if err := deploymentDS.UpsertDeployment(ctx, deployment); err != nil {
			t.Fatalf("Failed to create deployment %d: %v", i, err)
		}

		// Create process indicators for this deployment
		for j := 0; j < config.NumPodsPerDeployment; j++ {
			podID := uuid.NewV4().String()
			containerID := uuid.NewV4().String()

			for k := 0; k < config.NumProcessesPerPod; k++ {
				indicator := &storage.ProcessIndicator{
					Id:            uuid.NewV4().String(),
					DeploymentId:  deploymentID,
					PodId:         podID,
					ContainerName: "main-container",
					ClusterId:     clusterID,
					Namespace:     deployment.Namespace,
					Signal: &storage.ProcessSignal{
						Id:           uuid.NewV4().String(),
						ContainerId:  containerID,
						Time:         protocompat.TimestampNow(),
						Name:         fmt.Sprintf("process-%d", k),
						ExecFilePath: fmt.Sprintf("/usr/bin/process-%d", k%10),
						Args:         fmt.Sprintf("--arg1=value%d --arg2=value%d", k, j),
						Pid:          uint32(1000 + k),
						Uid:          0,
						Gid:          0,
						LineageInfo: []*storage.ProcessSignal_LineageInfo{
							{
								ParentExecFilePath: "/usr/bin/init",
								ParentUid:          0,
							},
						},
					},
				}

				if err := processDS.AddProcessIndicators(ctx, indicator); err != nil {
					t.Fatalf("Failed to create process indicator %d-%d-%d: %v", i, j, k, err)
				}
			}
		}
	}
}

// TestSeedBenchmarkData seeds the database with test data for benchmarking.
// Run this first to create a database, then set BENCHMARK_PRESEEDED_DB to the database name
// and run the benchmark tests.
//
// Example usage:
//
//	go test -run TestSeedBenchmarkData
//	# Note the database name from the output
//	BENCHMARK_PRESEEDED_DB=<db_name> go test -bench=BenchmarkBuildIndicatorFilter -memprofile=mem.prof
func TestSeedBenchmarkData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data seeding in short mode")
	}

	ctx := sac.WithAllAccess(context.Background())

	// Create a database with a known name for reuse
	dbName := "benchmark_lifecycle_preseeded"
	pgtest.DropDatabase(t, dbName)
	pgtest.CreateDatabase(t, dbName)

	pool := pgtest.ForTCustomPool(t, dbName)
	defer pool.Close()

	// Create schema
	gormDB := pgtest.OpenGormDB(t, pgtest.GetConnectionStringWithDatabaseName(t, dbName))
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	pgtest.CloseGormDB(t, gormDB)

	// Create datastores
	deploymentDS, err := deploymentDatastore.GetTestPostgresDataStore(t, pool)
	require.NoError(t, err)
	processDS := processIndicatorDatastore.GetTestPostgresDataStore(t, pool)
	require.NoError(t, err)

	// Seed with a large dataset
	config := BenchmarkConfig{
		NumDeployments:       100,
		NumPodsPerDeployment: 10,
		NumProcessesPerPod:   30,
	}

	t.Logf("Seeding database '%s' with %d deployments, %d pods per deployment, %d processes per pod",
		dbName, config.NumDeployments, config.NumPodsPerDeployment, config.NumProcessesPerPod)

	seedBenchmarkData(ctx, t, deploymentDS, processDS, config)

	totalProcesses := config.NumDeployments * config.NumPodsPerDeployment * config.NumProcessesPerPod
	t.Logf("Successfully seeded %d total process indicators", totalProcesses)
	t.Logf("To use this database for benchmarks, set: BENCHMARK_PRESEEDED_DB=%s", dbName)
	t.Logf("Example: BENCHMARK_PRESEEDED_DB=%s go test -bench=BenchmarkBuildIndicatorFilter -memprofile=mem.prof", dbName)
}

// BenchmarkBuildIndicatorFilter benchmarks the buildIndicatorFilter function
// using a pre-seeded database to avoid data seeding noise in memory profiles.
//
// Usage:
//  1. Run TestSeedBenchmarkData to create and seed the database
//  2. Set BENCHMARK_PRESEEDED_DB environment variable to the database name
//  3. Run this benchmark
func BenchmarkBuildIndicatorFilter(b *testing.B) {
	dbName := os.Getenv(preSeededDBEnvVar)
	if dbName == "" {
		b.Skipf("Set %s environment variable to run this benchmark with pre-seeded data", preSeededDBEnvVar)
	}

	ctx := sac.WithAllAccess(context.Background())

	// Connect to pre-seeded database
	pool := pgtest.ForTCustomPool(b, dbName)
	defer pool.Close()

	// Create datastores
	deploymentDS, err := deploymentDatastore.GetTestPostgresDataStore(b, pool)
	require.NoError(b, err)
	processDS := processIndicatorDatastore.GetTestPostgresDataStore(b, pool)
	require.NoError(b, err)

	// Count the data to verify
	deploymentIDs, err := deploymentDS.GetDeploymentIDs(ctx)
	require.NoError(b, err)
	b.Logf("Using pre-seeded database with %d deployments", len(deploymentIDs))

	// Create manager
	manager := &managerImpl{
		deploymentDataStore: deploymentDS,
		processesDataStore:  processDS,
		processFilter: filter.NewFilter(
			1000,                      // maxExactPathMatches
			10000,                     // maxUniqueProcesses
			[]int{100, 50, 25, 10, 5}, // fanOut
		),
	}

	// Reset timer to exclude setup time
	b.ResetTimer()
	// Reset filter for each iteration to get consistent measurements
	manager.processFilter = filter.NewFilter(
		1000,
		10000,
		[]int{100, 50, 25, 10, 5},
	)

	manager.buildIndicatorFilter()

}

// BenchmarkBuildIndicatorFilterOnce runs buildIndicatorFilter exactly once
// for clean memory profiling without iteration noise.
//
// Usage:
//  1. Run TestSeedBenchmarkData to create and seed the database
//  2. BENCHMARK_PRESEEDED_DB=<db_name> go test -bench=BenchmarkBuildIndicatorFilterOnce -benchtime=1x -memprofile=mem.prof
func BenchmarkBuildIndicatorFilterOnce(b *testing.B) {
	dbName := os.Getenv(preSeededDBEnvVar)
	if dbName == "" {
		b.Skipf("Set %s environment variable to run this benchmark with pre-seeded data", preSeededDBEnvVar)
	}

	ctx := sac.WithAllAccess(context.Background())
	// Connect to pre-seeded database
	pool := pgtest.ForTCustomPool(b, dbName)
	defer pool.Close()

	// Create datastores
	deploymentDS, err := deploymentDatastore.GetTestPostgresDataStore(b, pool)
	require.NoError(b, err)
	processDS := processIndicatorDatastore.GetTestPostgresDataStore(b, pool)
	require.NoError(b, err)

	// Count the data to verify
	deploymentIDs, err := deploymentDS.GetDeploymentIDs(ctx)
	require.NoError(b, err)
	b.Logf("Using pre-seeded database with %d deployments", len(deploymentIDs))

	// Create manager
	manager := &managerImpl{
		deploymentDataStore: deploymentDS,
		processesDataStore:  processDS,
		processFilter: filter.NewFilter(
			1000,                      // maxExactPathMatches
			10000,                     // maxUniqueProcesses
			[]int{100, 50, 25, 10, 5}, // fanOut
		),
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Run exactly once (use -benchtime=1x)
	for i := 0; i < b.N; i++ {
		manager.processFilter = filter.NewFilter(
			1000,
			10000,
			[]int{100, 50, 25, 10, 5},
		)

		manager.buildIndicatorFilter()
	}

	for i := 0; i < 10; i++ {
		runtime.GC()
		time.Sleep(500 * time.Millisecond)
	}
	f, err := os.Create("right.prof")
	if err != nil {
		b.Fatal(err)
	}

	defer f.Close()

	err = pprof.Lookup("heap").WriteTo(f, 0) // The 0 is for the default sampling rate
	if err != nil {
		b.Fatalf("could not write memory profile: %v", err)
	}

	fmt.Println(manager.processFilter)
}
