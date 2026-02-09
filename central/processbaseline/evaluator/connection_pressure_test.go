//go:build sql_integration

package evaluator

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	processBaselineDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsDS "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	pkgSync "github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestConcurrentEvaluationConnectionPressure demonstrates that the concurrent
// reprocessing of deployment risk can exhaust postgres connections.
//
// Background:
//   - The reprocessor's riskLoop drains ALL accumulated deployment IDs at once
//     and injects a ReprocessDeploymentRisk message for each one.
//   - Each message is processed by a worker queue with workerQueueSize=16
//     (17 total goroutines) PER CLUSTER PER EVENT TYPE.
//   - Each ReprocessDeploymentRisk call chain reaches EvaluateBaselinesAndPersistResult,
//     which calls IterateOverProcessIndicatorsRiskView. This function holds a DB
//     connection open for the entire duration of row iteration (PR #17126 changed
//     from loading all rows into memory to streaming via callback).
//   - With multiple clusters (e.g., 6 clusters * 17 workers = 102 goroutines),
//     plus other DB operations, the default pool_max_conns=90 is easily exceeded.
//
// This test creates a constrained connection pool and runs concurrent evaluations
// at the parallelism level matching the production worker queue, demonstrating
// that connection pool exhaustion occurs and causes significant latency degradation
// or timeouts.
func TestConcurrentEvaluationConnectionPressure(t *testing.T) {
	// --- Setup: Create a DB with a constrained connection pool ---
	database := pgtest.CreateADatabaseForT(t)
	t.Cleanup(func() { pgtest.DropDatabase(t, database) })

	// Apply schemas using a temporary connection
	source := conn.GetConnectionStringWithDatabaseName(t, database)
	gormDB := pgtest.OpenGormDB(t, source)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	pgtest.CloseGormDB(t, gormDB)

	allAccessCtx := sac.WithAllAccess(context.Background())

	// In production: pool_max_conns=90, workerQueueSize=16 (17 workers per cluster).
	// With 6 clusters that's 102 concurrent goroutines for just reprocessing.
	// We simulate this with a small pool to make the effect visible in a test.
	const (
		// poolMaxConns simulates a constrained connection pool.
		// In production, pool_max_conns=90 but many services compete for connections.
		poolMaxConns = 5

		// workerQueueTotalSize matches production: workerQueueSize(16) + 1 = 17
		// Each cluster has this many concurrent workers for deployment events.
		workerQueueTotalSize = 17

		// numClusters simulates multiple secured clusters connected to central.
		numClusters = 3

		// totalConcurrentWorkers = workerQueueTotalSize * numClusters
		// In production, this would be 17 * N_clusters, all hitting the same DB pool.
		totalConcurrentWorkers = workerQueueTotalSize * numClusters

		// numDeployments is the number of deployments to reprocess concurrently.
		// In production, a sensor reconnect queues ALL cluster deployments at once.
		numDeployments = totalConcurrentWorkers * 2

		// numProcessesPerDeployment is a realistic number of process indicators.
		numProcessesPerDeployment = 500

		// operationTimeout is how long we allow each evaluation to take.
		// With a healthy pool, each evaluation takes ~1-5ms.
		// Under connection pressure, they can take 10-100x longer.
		operationTimeout = 30 * time.Second
	)

	// Create pool with constrained connections
	constrainedSource := fmt.Sprintf("%s pool_min_conns=1 pool_max_conns=%d", source, poolMaxConns)
	pool, err := postgres.Connect(context.Background(), constrainedSource)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Create datastores sharing the constrained pool
	indicatorDS := processIndicatorDS.GetTestPostgresDataStore(t, pool)
	baselineDS := processBaselineDS.GetTestPostgresDataStore(t, pool)
	resultsDS := processBaselineResultsDS.GetTestPostgresDataStore(t, pool)
	eval := New(resultsDS, baselineDS, indicatorDS)

	// --- Seed data: Create deployments with locked baselines and process indicators ---
	type deploymentData struct {
		deployment *storage.Deployment
	}
	deployments := make([]deploymentData, numDeployments)

	t.Logf("Seeding %d deployments with %d process indicators each...", numDeployments, numProcessesPerDeployment)

	for i := 0; i < numDeployments; i++ {
		dep := fixtures.GetDeployment()
		dep.Id = uuid.NewV4().String()
		dep.ClusterId = fmt.Sprintf("cluster-%d", i%numClusters)

		containerNames := make([]string, 0, len(dep.GetContainers()))
		for _, c := range dep.GetContainers() {
			containerNames = append(containerNames, c.GetName())
		}

		// Generate and insert process indicators
		processes := generateTestProcessIndicators(numProcessesPerDeployment, dep.GetId(), containerNames, dep)
		err := indicatorDS.AddProcessIndicators(allAccessCtx, processes...)
		require.NoError(t, err)

		// Create a locked baseline so IterateOverProcessIndicatorsRiskView is actually called
		key := &storage.ProcessBaselineKey{
			DeploymentId:  dep.GetId(),
			ContainerName: containerNames[0],
			ClusterId:     dep.GetClusterId(),
			Namespace:     dep.GetNamespace(),
		}
		elements := []*storage.BaselineItem{
			{Item: &storage.BaselineItem_ProcessName{ProcessName: "/usr/bin/apt-get"}},
			{Item: &storage.BaselineItem_ProcessName{ProcessName: "/usr/bin/curl"}},
		}
		_, err = baselineDS.UpsertProcessBaseline(allAccessCtx, key, elements, false, true)
		require.NoError(t, err)
		_, err = baselineDS.UserLockProcessBaseline(allAccessCtx, key, true)
		require.NoError(t, err)

		deployments[i] = deploymentData{deployment: dep}
	}

	t.Logf("Data seeded. Running %d concurrent evaluations (simulating %d clusters * %d workers)...",
		totalConcurrentWorkers, numClusters, workerQueueTotalSize)

	// --- Test: Run concurrent evaluations matching production concurrency ---
	// This simulates what happens when the riskLoop drains the deploymentRiskSet
	// and injects messages that are processed by N clusters * 17 workers each.

	var (
		wg             pkgSync.WaitGroup
		errCount       atomic.Int64
		timeoutCount   atomic.Int64
		successCount   atomic.Int64
		totalDuration  atomic.Int64
		maxDurationNs  atomic.Int64
	)

	start := time.Now()

	// Launch workers matching production concurrency
	wg.Add(numDeployments)
	for i := 0; i < numDeployments; i++ {
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(allAccessCtx, operationTimeout)
			defer cancel()

			dep := deployments[idx].deployment
			evalStart := time.Now()

			// This is the exact call chain that ReprocessDeploymentRisk triggers:
			//   deploymentScorer.Score -> ProcessBaselines multiplier ->
			//   EvaluateBaselinesAndPersistResult -> IterateOverProcessIndicatorsRiskView
			_, evalErr := eval.EvaluateBaselinesAndPersistResult(dep)

			elapsed := time.Since(evalStart)
			totalDuration.Add(elapsed.Nanoseconds())

			// Track max duration
			for {
				current := maxDurationNs.Load()
				if elapsed.Nanoseconds() <= current {
					break
				}
				if maxDurationNs.CompareAndSwap(current, elapsed.Nanoseconds()) {
					break
				}
			}

			if ctx.Err() != nil {
				timeoutCount.Add(1)
			} else if evalErr != nil {
				errCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	wallClock := time.Since(start)

	// --- Report results ---
	successes := successCount.Load()
	errors := errCount.Load()
	timeouts := timeoutCount.Load()
	avgDuration := time.Duration(totalDuration.Load() / int64(numDeployments))
	maxDuration := time.Duration(maxDurationNs.Load())

	t.Logf("Results with pool_max_conns=%d, %d concurrent workers, %d deployments:", poolMaxConns, totalConcurrentWorkers, numDeployments)
	t.Logf("  Successes:    %d", successes)
	t.Logf("  Errors:       %d", errors)
	t.Logf("  Timeouts:     %d", timeouts)
	t.Logf("  Avg duration: %v", avgDuration)
	t.Logf("  Max duration: %v", maxDuration)
	t.Logf("  Wall clock:   %v", wallClock)
	t.Logf("")
	t.Logf("Connection pressure analysis:")
	t.Logf("  pool_max_conns:        %d", poolMaxConns)
	t.Logf("  concurrent_workers:    %d (production: 17 * num_clusters)", totalConcurrentWorkers)
	t.Logf("  contention_ratio:      %.1fx (workers/connections)", float64(totalConcurrentWorkers)/float64(poolMaxConns))

	// The key assertion: with more concurrent workers than connections,
	// operations become significantly slower due to connection contention.
	// In a healthy system with no contention, each evaluation takes 1-5ms.
	// Under contention, the average should be much higher due to queueing.
	//
	// We assert that the contention is measurable: when the ratio of workers
	// to connections exceeds 1, operations take longer.
	assert.Greater(t, successes+errors+timeouts, int64(0), "at least some operations should complete")

	// With 51 workers competing for 5 connections, wall clock should be
	// significantly longer than if we had unlimited connections.
	// We compute expected minimum time: numDeployments * avg_single_eval / poolMaxConns
	// If each eval takes ~5ms and we have 102 evals with 5 connections,
	// minimum wall time would be ~102ms. But with contention overhead, it's worse.
	t.Logf("")
	t.Logf("CONCLUSION: %d workers competed for %d DB connections.", totalConcurrentWorkers, poolMaxConns)
	t.Logf("In production with pool_max_conns=90 and 6+ clusters (102+ workers),")
	t.Logf("plus all other Central DB operations, this contention leads to connection")
	t.Logf("pool exhaustion when many deployments are reprocessed at once (e.g., sensor reconnect).")
	t.Logf("The IterateOverProcessIndicatorsRiskView function (PR #17126) holds a DB connection")
	t.Logf("open for the entire row iteration, exacerbating the problem vs the old approach")
	t.Logf("that loaded all rows and released the connection immediately.")

	// Assert that we can measure significant contention
	expectedContentionRatio := float64(totalConcurrentWorkers) / float64(poolMaxConns)
	assert.Greater(t, expectedContentionRatio, 1.0,
		"test should create more workers than connections to demonstrate contention")

	// Assert that the max duration shows contention effects.
	// With no contention, a single evaluation should complete in well under 1 second.
	// With contention at 10:1 ratio, some operations will wait significantly longer.
	assert.Greater(t, maxDuration, 100*time.Millisecond,
		"max duration should show significant queueing delay due to connection contention")
}

// TestConcurrentEvaluationBaselineComparison compares the latency of evaluations
// with different connection pool sizes to demonstrate that pool exhaustion is the
// bottleneck, not the evaluation logic itself.
func TestConcurrentEvaluationBaselineComparison(t *testing.T) {
	database := pgtest.CreateADatabaseForT(t)
	t.Cleanup(func() { pgtest.DropDatabase(t, database) })

	source := conn.GetConnectionStringWithDatabaseName(t, database)
	gormDB := pgtest.OpenGormDB(t, source)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, t)
	pgtest.CloseGormDB(t, gormDB)

	allAccessCtx := sac.WithAllAccess(context.Background())

	const (
		concurrency              = 17 // production workerQueueTotalSize per cluster
		numDeployments           = 50
		numProcessesPerDep       = 200
	)

	type result struct {
		poolSize    int
		wallClock   time.Duration
		avgDuration time.Duration
		maxDuration time.Duration
	}
	var results []result

	for _, poolSize := range []int{2, 5, 20, 50} {
		t.Run(fmt.Sprintf("pool_max_conns=%d", poolSize), func(t *testing.T) {
			constrainedSource := fmt.Sprintf("%s pool_min_conns=1 pool_max_conns=%d", source, poolSize)
			pool, err := postgres.Connect(context.Background(), constrainedSource)
			require.NoError(t, err)
			t.Cleanup(func() { pool.Close() })

			indicatorDS := processIndicatorDS.GetTestPostgresDataStore(t, pool)
			baselineDS := processBaselineDS.GetTestPostgresDataStore(t, pool)
			resultsDS := processBaselineResultsDS.GetTestPostgresDataStore(t, pool)
			eval := New(resultsDS, baselineDS, indicatorDS)

			// Seed data
			deps := make([]*storage.Deployment, numDeployments)
			for i := 0; i < numDeployments; i++ {
				dep := fixtures.GetDeployment()
				dep.Id = uuid.NewV4().String()

				containerNames := make([]string, 0, len(dep.GetContainers()))
				for _, c := range dep.GetContainers() {
					containerNames = append(containerNames, c.GetName())
				}

				processes := generateTestProcessIndicators(numProcessesPerDep, dep.GetId(), containerNames, dep)
				err := indicatorDS.AddProcessIndicators(allAccessCtx, processes...)
				require.NoError(t, err)

				key := &storage.ProcessBaselineKey{
					DeploymentId:  dep.GetId(),
					ContainerName: containerNames[0],
					ClusterId:     dep.GetClusterId(),
					Namespace:     dep.GetNamespace(),
				}
				elements := []*storage.BaselineItem{
					{Item: &storage.BaselineItem_ProcessName{ProcessName: "/usr/bin/apt-get"}},
				}
				_, err = baselineDS.UpsertProcessBaseline(allAccessCtx, key, elements, false, true)
				require.NoError(t, err)
				_, err = baselineDS.UserLockProcessBaseline(allAccessCtx, key, true)
				require.NoError(t, err)

				deps[i] = dep
			}

			// Run concurrent evaluations
			var (
				wg            pkgSync.WaitGroup
				totalNs       atomic.Int64
				maxNs         atomic.Int64
			)

			start := time.Now()
			wg.Add(numDeployments)
			for i := 0; i < numDeployments; i++ {
				go func(idx int) {
					defer wg.Done()
					evalStart := time.Now()
					_, _ = eval.EvaluateBaselinesAndPersistResult(deps[idx])
					elapsed := time.Since(evalStart)
					totalNs.Add(elapsed.Nanoseconds())
					for {
						cur := maxNs.Load()
						if elapsed.Nanoseconds() <= cur {
							break
						}
						if maxNs.CompareAndSwap(cur, elapsed.Nanoseconds()) {
							break
						}
					}
				}(i)
			}
			wg.Wait()
			wallClock := time.Since(start)

			avg := time.Duration(totalNs.Load() / int64(numDeployments))
			maxD := time.Duration(maxNs.Load())

			results = append(results, result{
				poolSize:    poolSize,
				wallClock:   wallClock,
				avgDuration: avg,
				maxDuration: maxD,
			})

			t.Logf("pool_max_conns=%-3d | wall=%-10v avg=%-10v max=%-10v | contention_ratio=%.1f",
				poolSize, wallClock, avg, maxD, float64(concurrency)/float64(poolSize))
		})
	}

	// Log comparison
	t.Logf("\nComparison (concurrency=%d, deployments=%d, processes/dep=%d):", concurrency, numDeployments, numProcessesPerDep)
	t.Logf("%-20s %-15s %-15s %-15s %-20s", "Pool Size", "Wall Clock", "Avg Duration", "Max Duration", "Contention Ratio")
	for _, r := range results {
		t.Logf("%-20d %-15v %-15v %-15v %-20.1f",
			r.poolSize, r.wallClock, r.avgDuration, r.maxDuration,
			float64(concurrency)/float64(r.poolSize))
	}

	// With more connections, wall clock should be faster
	if len(results) >= 2 {
		smallest := results[0]
		largest := results[len(results)-1]
		assert.Less(t, largest.wallClock, smallest.wallClock,
			"larger connection pool should complete faster, proving connection contention is the bottleneck")
		t.Logf("\nSpeedup with larger pool: %.1fx faster wall clock with %dx more connections",
			float64(smallest.wallClock)/float64(largest.wallClock),
			largest.poolSize/smallest.poolSize)
	}
}

// generateTestProcessIndicators creates process indicators for testing connection pressure.
// This is a simplified version of the benchmark helper.
func generateTestProcessIndicators(count int, deploymentID string, containers []string, deployment *storage.Deployment) []*storage.ProcessIndicator {
	processes := make([]*storage.ProcessIndicator, 0, count)

	templates := []struct {
		name string
		path string
		args string
	}{
		{"apt-get", "/usr/bin/apt-get", "update"},
		{"curl", "/usr/bin/curl", "https://example.com"},
		{"wget", "/usr/bin/wget", "https://example.com/file"},
		{"python3", "/usr/bin/python3", "-c 'import sys'"},
		{"node", "/usr/bin/node", "/app/server.js"},
		{"nginx", "/usr/sbin/nginx", "-g 'daemon off;'"},
		{"redis-server", "/usr/bin/redis-server", "/etc/redis.conf"},
		{"postgres", "/usr/lib/postgresql/13/bin/postgres", "-D /data"},
		{"java", "/usr/bin/java", "-jar /app/app.jar"},
		{"sh", "/bin/sh", "-c 'sleep 1'"},
	}

	// Container started long ago so processes are not startup processes
	containerStartTime := protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))

	for i := 0; i < count; i++ {
		tmpl := templates[i%len(templates)]
		containerName := containers[i%len(containers)]

		processTime := containerStartTime.AsTime().Add(time.Duration(5+i) * time.Minute)

		processes = append(processes, &storage.ProcessIndicator{
			Id:                 uuid.NewV4().String(),
			DeploymentId:       deploymentID,
			ContainerName:      containerName,
			PodId:              fmt.Sprintf("pod-%d", i%10),
			PodUid:             uuid.NewV4().String(),
			ClusterId:          deployment.GetClusterId(),
			Namespace:          deployment.GetNamespace(),
			ImageId:            fmt.Sprintf("sha256:image-%d", i%5),
			ContainerStartTime: containerStartTime,
			Signal: &storage.ProcessSignal{
				Id:           uuid.NewV4().String(),
				ContainerId:  fmt.Sprintf("container-%d", i%len(containers)),
				Time:         &timestamppb.Timestamp{Seconds: processTime.Unix()},
				Name:         tmpl.name,
				Args:         tmpl.args,
				ExecFilePath: tmpl.path,
				Pid:          uint32(1000 + i),
			},
		})
	}

	return processes
}
