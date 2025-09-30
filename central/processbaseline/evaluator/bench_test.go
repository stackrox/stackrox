//go:build sql_integration

package evaluator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	processBaselineDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsDS "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testCtx = sac.WithAllAccess(context.Background())

// generateProcessIndicators creates realistic test process indicators
func generateProcessIndicators(numProcesses int, deploymentID string, containers []string, withLargeArgs bool, deployment *storage.Deployment) []*storage.ProcessIndicator {
	processes := make([]*storage.ProcessIndicator, 0, numProcesses)

	// Common process templates
	processTemplates := []struct {
		name string
		path string
		args string
	}{
		{"apt-get", "/usr/bin/apt-get", "update"},
		{"curl", "/usr/bin/curl", "https://example.com/api/data"},
		{"wget", "/usr/bin/wget", "https://download.example.com/file.tar.gz"},
		{"python3", "/usr/bin/python3", "-c 'import sys; print(sys.version)'"},
		{"node", "/usr/bin/node", "/app/server.js --port=3000"},
		{"nginx", "/usr/sbin/nginx", "-g 'daemon off;'"},
		{"redis-server", "/usr/bin/redis-server", "/etc/redis/redis.conf"},
		{"postgres", "/usr/lib/postgresql/13/bin/postgres", "-D /var/lib/postgresql/data"},
		{"java", "/usr/bin/java", "-Xmx2g -jar /app/application.jar"},
		{"sh", "/bin/sh", "-c 'while true; do sleep 1; done'"},
	}

	// Create varied container start times to simulate realistic startup scenarios
	// Some containers started recently (have startup processes), others started long ago
	containerStartTimes := make(map[string]*timestamppb.Timestamp)

	for i, containerName := range containers {
		var startTime time.Time
		switch i % 4 {
		case 0:
			// 25% of containers: started 10 seconds ago (most processes will be startup)
			startTime = time.Now().Add(-10 * time.Second)
		case 1:
			// 25% of containers: started 30 seconds ago (some startup processes)
			startTime = time.Now().Add(-30 * time.Second)
		case 2:
			// 25% of containers: started 2 minutes ago (no startup processes)
			startTime = time.Now().Add(-2 * time.Minute)
		case 3:
			// 25% of containers: started 1 hour ago (no startup processes)
			startTime = time.Now().Add(-1 * time.Hour)
		}
		containerStartTimes[containerName] = protoconv.MustConvertTimeToTimestamp(startTime)
	}

	for i := 0; i < numProcesses; i++ {
		template := processTemplates[i%len(processTemplates)]
		containerName := containers[i%len(containers)]

		args := template.args
		if withLargeArgs {
			// Create large argument strings to simulate real-world scenarios
			longArg := strings.Repeat(fmt.Sprintf("--config-param-%d=very-long-configuration-value-%d ", i, i), 10)
			args = template.args + " " + longArg
		}

		// Vary process execution times relative to container start
		containerStartTime := containerStartTimes[containerName]
		var processTime time.Time

		// Create a realistic distribution of process start times
		switch i % 10 {
		case 0, 1:
			// 20% of processes: started 5-15 seconds after container (likely startup)
			processTime = containerStartTime.AsTime().Add(time.Duration(5+i%10) * time.Second)
		case 2, 3:
			// 20% of processes: started 30-50 seconds after container (maybe startup)
			processTime = containerStartTime.AsTime().Add(time.Duration(30+i%20) * time.Second)
		default:
			// 60% of processes: started 2-30 minutes after container (not startup)
			processTime = containerStartTime.AsTime().Add(time.Duration(2+i%28) * time.Minute)
		}

		process := &storage.ProcessIndicator{
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
				Time:         protoconv.MustConvertTimeToTimestamp(processTime),
				Name:         template.name,
				Args:         args,
				ExecFilePath: template.path,
				Pid:          uint32(1000 + i),
				Uid:          0,
				Gid:          0,
				Scraped:      false,
				LineageInfo: []*storage.ProcessSignal_LineageInfo{
					{
						ParentUid:          0,
						ParentExecFilePath: "/sbin/init",
					},
				},
			},
		}

		processes = append(processes, process)
	}

	return processes
}

// BenchmarkEvaluateBaselinesAndPersistResult benchmarks the main evaluation function with real database
func BenchmarkEvaluateBaselinesAndPersistResult(b *testing.B) {
	deployment := fixtures.GetDeployment()
	deploymentID := uuid.NewV4().String()
	deployment.Id = deploymentID

	// Test scenarios with different scales
	scenarios := []struct {
		name          string
		numProcesses  int
		numContainers int
		withLargeArgs bool
	}{
		{"100_processes_2_containers", 100, 2, false},
		{"500_processes_3_containers", 500, 3, false},
		{"1000_processes_5_containers", 1000, 5, false},
		{"1000_processes_5_containers_large_args", 1000, 5, true},
		{"2000_processes_10_containers", 2000, 10, false},
		{"10000_processes_20_containers", 10000, 20, false},
		{"25000_processes_50_containers", 25000, 50, false},
		{"50000_processes_100_containers", 50000, 100, false},
		{"50000_processes_100_containers_large_args", 50000, 100, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			processIndicatorDatastore, processBaselineDatastore, processBaselineResultsDatastore := setupStores(b)
			evaluator := New(processBaselineResultsDatastore, processBaselineDatastore, processIndicatorDatastore)

			// Use actual container names from the deployment
			containerNames := make([]string, 0, len(deployment.GetContainers()))
			for _, container := range deployment.GetContainers() {
				containerNames = append(containerNames, container.GetName())
			}

			// Ensure we have enough containers for the test
			for len(containerNames) < scenario.numContainers {
				containerNames = append(containerNames, fmt.Sprintf("extra-container-%d", len(containerNames)))
			}

			// Generate and insert process indicators
			processes := generateProcessIndicators(scenario.numProcesses, deploymentID, containerNames, scenario.withLargeArgs, deployment)
			err := processIndicatorDatastore.AddProcessIndicators(testCtx, processes...)
			require.NoError(b, err)

			// Create baselines - first container has locked baseline with only SOME processes
			// This ensures we get violations from processes NOT in the baseline
			// NOTE: These must match the ExecFilePath from processTemplates
			baselineProcesses := []string{"/usr/bin/apt-get", "/usr/bin/curl"} // Only 2 out of 10 process types

			// Create locked baseline for first container
			key := &storage.ProcessBaselineKey{
				DeploymentId:  deploymentID,
				ContainerName: containerNames[0],
				ClusterId:     deployment.GetClusterId(), // Use deployment's cluster ID
				Namespace:     deployment.GetNamespace(), // Use deployment's namespace
			}
			elements := make([]*storage.BaselineItem, len(baselineProcesses))
			for i, processPath := range baselineProcesses {
				elements[i] = &storage.BaselineItem{
					Item: &storage.BaselineItem_ProcessName{
						ProcessName: processPath,
					},
				}
			}
			_, err = processBaselineDatastore.UpsertProcessBaseline(testCtx, key, elements, false, true)
			require.NoError(b, err)

			// We need to actually lock the baseline
			baseline, err := processBaselineDatastore.UserLockProcessBaseline(testCtx, key, true)
			require.NoError(b, err)
			require.NotNil(b, baseline)

			b.ResetTimer()

			// Run the benchmark
			for i := 0; i < b.N; i++ {
				violatingProcesses, err := evaluator.EvaluateBaselinesAndPersistResult(deployment)
				require.NoError(b, err)

				// Ensure we have realistic violation patterns
				_ = violatingProcesses // Startup processes are filtered out by evaluator
			}
		})
	}
}

// BenchmarkEvaluateBaselinesSmallScale benchmarks the evaluator with a smaller dataset
func BenchmarkEvaluateBaselinesSmallScale(b *testing.B) {
	processIndicatorDatastore, processBaselineDatastore, processBaselineResultsDatastore := setupStores(b)

	deployment := fixtures.GetDeployment()
	deploymentID := uuid.NewV4().String()
	deployment.Id = deploymentID

	// Use actual container names from the deployment
	containerNames := make([]string, 0, len(deployment.GetContainers()))
	for _, container := range deployment.GetContainers() {
		containerNames = append(containerNames, container.GetName())
	}

	// Ensure we have enough containers for the test
	for len(containerNames) < 5 {
		containerNames = append(containerNames, fmt.Sprintf("extra-container-%d", len(containerNames)))
	}

	// Generate and insert process indicators
	processes := generateProcessIndicators(2500, deploymentID, containerNames, false, deployment)
	err := processIndicatorDatastore.AddProcessIndicators(testCtx, processes...)
	require.NoError(b, err)

	// Create locked baseline for first container
	baselineProcesses := []string{"/usr/bin/apt-get", "/usr/bin/curl"}
	key := &storage.ProcessBaselineKey{
		DeploymentId:  deploymentID,
		ContainerName: containerNames[0],
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
	}
	elements := make([]*storage.BaselineItem, len(baselineProcesses))
	for i, processPath := range baselineProcesses {
		elements[i] = &storage.BaselineItem{
			Item: &storage.BaselineItem_ProcessName{
				ProcessName: processPath,
			},
		}
	}
	_, err = processBaselineDatastore.UpsertProcessBaseline(testCtx, key, elements, false, true)
	require.NoError(b, err)

	// We need to actually lock the baseline
	baseline, err := processBaselineDatastore.UserLockProcessBaseline(testCtx, key, true)
	require.NoError(b, err)
	require.NotNil(b, baseline)

	b.Run("2500_processes_5_containers", func(b *testing.B) {
		evaluator := New(processBaselineResultsDatastore, processBaselineDatastore, processIndicatorDatastore)

		for i := 0; i < b.N; i++ {
			violatingProcesses, err := evaluator.EvaluateBaselinesAndPersistResult(deployment)
			require.NoError(b, err)
			_ = violatingProcesses // May be nil if no violations found
		}
	})

}

func setupStores(b *testing.B) (processIndicatorDS.DataStore, processBaselineDS.DataStore, processBaselineResultsDS.DataStore) {
	testDB := pgtest.ForT(b)

	processIndicatorDatastore := processIndicatorDS.GetTestPostgresDataStore(b, testDB.DB)
	processBaselineDatastore := processBaselineDS.GetTestPostgresDataStore(b, testDB.DB)
	processBaselineResultsDatastore := processBaselineResultsDS.GetTestPostgresDataStore(b, testDB.DB)
	return processIndicatorDatastore, processBaselineDatastore, processBaselineResultsDatastore
}
