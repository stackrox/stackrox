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
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// BenchmarkConfig holds configuration for benchmark tests
type BenchmarkConfig struct {
	NumDeployments       int
	NumPodsPerDeployment int
	NumProcessesPerPod   int
}

// defaultBenchmarkConfig is the standard configuration used for all benchmarks
var defaultBenchmarkConfig = BenchmarkConfig{
	NumDeployments:       1000,
	NumPodsPerDeployment: 10,
	NumProcessesPerPod:   10,
} // 30,000 total processes

// benchmarkDBSetup holds the database setup for benchmarks
type benchmarkDBSetup struct {
	deploymentDS deploymentDatastore.DataStore
	processDS    processIndicatorDatastore.DataStore
	testDB       *pgtest.TestPostgres
}

// setupBenchmarkDB creates a fresh database, applies schema, and seeds test data.
// The setup time is excluded from benchmark measurements.
func setupBenchmarkDB(b *testing.B, config BenchmarkConfig) *benchmarkDBSetup {
	ctx := sac.WithAllAccess(context.Background())

	testDB := pgtest.ForT(b)
	deploymentDS, err := deploymentDatastore.GetTestPostgresDataStore(b, testDB)
	require.NoError(b, err)
	processDS := processIndicatorDatastore.GetTestPostgresDataStore(b, testDB)

	seedBenchmarkData(ctx, b, deploymentDS, processDS, config)

	return &benchmarkDBSetup{
		deploymentDS: deploymentDS,
		processDS:    processDS,
		testDB:       testDB,
	}
}

// seedBenchmarkData populates the database with test data
func seedBenchmarkData(
	ctx context.Context,
	tb testing.TB,
	deploymentDS deploymentDatastore.DataStore,
	processDS processIndicatorDatastore.DataStore,
	config BenchmarkConfig,
) {
	ctx = sac.WithAllAccess(ctx)
	clusterID := uuid.NewV4().String()

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
			tb.Fatalf("Failed to create deployment %d: %v", i, err)
		}

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
					Namespace:     deployment.GetNamespace(),
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
					tb.Fatalf("Failed to create process indicator %d-%d-%d: %v", i, j, k, err)
				}
			}
		}
	}
}

// createTestManager creates a manager instance with the standard filter configuration
func createTestManager(deploymentDS deploymentDatastore.DataStore, processDS processIndicatorDatastore.DataStore) *managerImpl {
	return &managerImpl{
		deploymentDataStore: deploymentDS,
		processesDataStore:  processDS,
		processFilter: filter.NewFilter(
			1000,
			10000,
			[]int{100, 50, 25, 10, 5},
		),
	}
}

// BenchmarkBuildIndicatorFilterPerformance benchmarks the buildIndicatorFilter function
// across multiple iterations to measure filter building performance.
// Usage:
//
//	go test -bench=BenchmarkBuildIndicatorFilterPerformance
func BenchmarkBuildIndicatorFilterPerformance(b *testing.B) {
	// Suppress logs for clean benchmark output
	logging.SetGlobalLogLevel(zapcore.PanicLevel)

	// Create and seed a fresh database (excluded from timing)
	setup := setupBenchmarkDB(b, defaultBenchmarkConfig)

	// Create manager
	manager := createTestManager(setup.deploymentDS, setup.processDS)

	// Start timing the actual benchmark
	for b.Loop() {
		// Reset filter to empty state before each iteration
		manager.processFilter = filter.NewFilter(
			1000,
			10000,
			[]int{100, 50, 25, 10, 5},
		)

		// Build the filter (this is what we're measuring)
		manager.buildIndicatorFilter()
	}

}

// BenchmarkBuildIndicatorFilterMemory benchmarks memory usage of buildIndicatorFilter.
// This benchmark runs buildIndicatorFilter exactly once and capture a memory profile of it.
// The profile is written to "indicator_filter_memory.prof" in the current directory.
// Usage:
//
//	go test -bench=BenchmarkBuildIndicatorFilterMemory
//	go tool pprof indicator_filter_memory.prof
func BenchmarkBuildIndicatorFilterMemory(b *testing.B) {
	// Note that we're not messing with timers like in above test
	// because the sole purpose of this test is to write a heap profile
	setup := setupBenchmarkDB(b, defaultBenchmarkConfig)
	defer setup.testDB.Close()
	// Create manager
	manager := createTestManager(setup.deploymentDS, setup.processDS)

	manager.processFilter = filter.NewFilter(
		1000,
		10000,
		[]int{100, 50, 25, 10, 5},
	)

	manager.buildIndicatorFilter()

	// Force multiple garbage collections to get accurate heap profile
	for i := 0; i < 10; i++ {
		runtime.GC()
		time.Sleep(500 * time.Millisecond)
	}

	// Write memory profile
	profileFile := "indicator_filter_memory.prof"
	f, err := os.Create(profileFile)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { utils.Must(f.Close()) })

	err = pprof.Lookup("heap").WriteTo(f, 0)
	if err != nil {
		b.Fatalf("could not write memory profile: %v", err)
	}

	// we need to reference manager here otherwise the GC will remove it before writing the profile
	b.Logf("%T", manager.processFilter)
}
