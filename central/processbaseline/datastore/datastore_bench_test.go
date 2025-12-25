//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

var benchmarkCtx = sac.WithAllAccess(context.Background())

// generateProcessBaselines creates realistic test process baselines
func generateProcessBaselines(numBaselines int, elementsPerBaseline int) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, 0, numBaselines)

	// Use fixture cluster IDs
	clusterIDs := []string{
		fixtureconsts.Cluster1,
		fixtureconsts.Cluster2,
		fixtureconsts.Cluster3,
	}

	// Common process templates to ensure some overlap across baselines
	processTemplates := []string{
		"/usr/bin/apt-get",
		"/usr/bin/curl",
		"/usr/bin/wget",
		"/usr/bin/python3",
		"/usr/bin/node",
		"/usr/sbin/nginx",
		"/usr/bin/redis-server",
		"/usr/lib/postgresql/13/bin/postgres",
		"/usr/bin/java",
		"/bin/sh",
		"/bin/bash",
		"/usr/bin/perl",
		"/usr/bin/ruby",
		"/usr/bin/php",
		"/usr/bin/git",
		"/usr/bin/docker",
		"/usr/bin/kubectl",
		"/usr/bin/helm",
		"/usr/bin/terraform",
		"/usr/bin/ansible",
	}

	// Use fixture deployment IDs to create multiple containers per deployment
	// This simulates realistic scenarios where a single deployment has multiple containers (e.g., frontend, backend, database)
	deploymentIDs := []string{
		fixtureconsts.Deployment1,
		fixtureconsts.Deployment2,
		fixtureconsts.Deployment3,
		fixtureconsts.Deployment4,
		fixtureconsts.Deployment5,
		fixtureconsts.Deployment6,
	}

	// Container names - 15 different container types, enabling multiple containers per deployment
	// With 15 containers and ~30% deployments, average deployment has ~3-4 containers
	containerNames := []string{
		"frontend", "backend", "database", "cache", "queue",
		"worker", "api", "web", "app", "service",
		"proxy", "gateway", "scheduler", "collector", "monitor",
	}

	// Create baselines with varied characteristics
	// Each deployment can have multiple containers
	for i := 0; i < numBaselines; i++ {
		deploymentID := deploymentIDs[i%len(deploymentIDs)]
		clusterID := clusterIDs[i%len(clusterIDs)]
		namespace := fmt.Sprintf("namespace-%d", i%20) // 20 different namespaces
		containerName := containerNames[i%len(containerNames)]

		key := &storage.ProcessBaselineKey{
			DeploymentId:  deploymentID,
			ContainerName: containerName,
			ClusterId:     clusterID,
			Namespace:     namespace,
		}

		// Create elements for this baseline
		elements := make([]*storage.BaselineElement, 0, elementsPerBaseline)
		for j := 0; j < elementsPerBaseline; j++ {
			// Mix common processes with unique ones
			var processName string
			if j < len(processTemplates) {
				processName = processTemplates[j]
			} else {
				processName = fmt.Sprintf("/usr/local/bin/custom-process-%d-%d", i, j)
			}

			elements = append(elements, &storage.BaselineElement{
				Element: &storage.BaselineItem{
					Item: &storage.BaselineItem_ProcessName{
						ProcessName: processName,
					},
				},
				Auto: j%2 == 0, // 50% auto-generated, 50% manual
			})
		}

		baseline := &storage.ProcessBaseline{
			Key:      key,
			Elements: elements,
		}

		baselines = append(baselines, baseline)
	}

	return baselines
}

// BenchmarkAddProcessBaseline benchmarks the Add operation with different scales
func BenchmarkAddProcessBaseline(b *testing.B) {
	scenarios := []struct {
		name                string
		elementsPerBaseline int
	}{
		{"50_elements", 50},
		{"500_elements", 500},
		{"1000_elements", 1000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			testDB := pgtest.ForT(b)
			datastore := GetTestPostgresDataStore(b, testDB.DB)

			// Pre-generate baselines to avoid measuring generation time
			baselines := generateProcessBaselines(b.N, scenario.elementsPerBaseline)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := datastore.AddProcessBaseline(benchmarkCtx, baselines[i])
				require.NoError(b, err)
			}
		})
	}
}

// BenchmarkSearchProcessBaseline benchmarks the Search operation with different dataset sizes and query types
func BenchmarkSearchProcessBaseline(b *testing.B) {
	scenarios := []struct {
		name                string
		numBaselines        int
		elementsPerBaseline int
	}{
		{"1000_baselines_50_elements", 1000, 50},
		{"5000_baselines_100_elements", 5000, 100},
		{"10000_baselines_50_elements", 10000, 50},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			testDB := pgtest.ForT(b)
			datastore := GetTestPostgresDataStore(b, testDB.DB)

			// Pre-populate the database
			baselines := generateProcessBaselines(scenario.numBaselines, scenario.elementsPerBaseline)
			for _, baseline := range baselines {
				_, err := datastore.AddProcessBaseline(benchmarkCtx, baseline)
				require.NoError(b, err)
			}

			// Test different query types focused on deployment ID (most common query pattern)
			// Use fixture deployment IDs that are guaranteed to exist in the dataset
			queries := []struct {
				name  string
				query *v1.Query
			}{
				{
					"match_single_deployment",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment1).
						ProtoQuery(),
				},
				{
					"match_deployment_with_container",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment1).
						AddExactMatches(pkgSearch.ContainerName, "frontend").
						ProtoQuery(),
				},
				{
					"match_multiple_deployments",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment1, fixtureconsts.Deployment2, fixtureconsts.Deployment3).
						ProtoQuery(),
				},
				{
					"match_container_name",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.ContainerName, "backend").
						ProtoQuery(),
				},
			}

			for _, query := range queries {
				b.Run(query.name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := datastore.Search(benchmarkCtx, query.query)
						require.NoError(b, err)
					}
				})
			}
		})
	}
}

// BenchmarkSearchRawProcessBaselines benchmarks the SearchRawProcessBaselines operation
func BenchmarkSearchRawProcessBaselines(b *testing.B) {
	scenarios := []struct {
		name                string
		numBaselines        int
		elementsPerBaseline int
	}{
		{"1000_baselines_50_elements", 1000, 50},
		{"5000_baselines_100_elements", 5000, 100},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			testDB := pgtest.ForT(b)
			datastore := GetTestPostgresDataStore(b, testDB.DB)

			// Pre-populate the database
			baselines := generateProcessBaselines(scenario.numBaselines, scenario.elementsPerBaseline)
			for _, baseline := range baselines {
				_, err := datastore.AddProcessBaseline(benchmarkCtx, baseline)
				require.NoError(b, err)
			}

			// Test different query types focused on deployment ID
			// Use fixture deployment IDs that are guaranteed to exist in the dataset
			queries := []struct {
				name  string
				query *v1.Query
			}{
				{
					"match_single_deployment",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment1).
						ProtoQuery(),
				},
				{
					"match_deployment_with_container",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment2).
						AddExactMatches(pkgSearch.ContainerName, "frontend").
						ProtoQuery(),
				},
				{
					"match_multiple_deployments",
					pkgSearch.NewQueryBuilder().
						AddExactMatches(pkgSearch.DeploymentID, fixtureconsts.Deployment1, fixtureconsts.Deployment2).
						ProtoQuery(),
				},
			}

			for _, query := range queries {
				b.Run(query.name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := datastore.SearchRawProcessBaselines(benchmarkCtx, query.query)
						require.NoError(b, err)
					}
				})
			}
		})
	}
}

// BenchmarkDeleteProcessBaseline benchmarks the Delete operation
func BenchmarkDeleteProcessBaseline(b *testing.B) {
	scenarios := []struct {
		name                string
		elementsPerBaseline int
	}{
		{"100_elements", 100},
		{"500_elements", 500},
		{"1000_elements", 1000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			testDB := pgtest.ForT(b)
			datastore := GetTestPostgresDataStore(b, testDB.DB)

			// Pre-populate the database with baselines
			baselines := generateProcessBaselines(b.N, scenario.elementsPerBaseline)
			keys := make([]*storage.ProcessBaselineKey, b.N)
			for i, baseline := range baselines {
				_, err := datastore.AddProcessBaseline(benchmarkCtx, baseline)
				require.NoError(b, err)
				keys[i] = baseline.GetKey()
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := datastore.RemoveProcessBaseline(benchmarkCtx, keys[i])
				require.NoError(b, err)
			}
		})
	}
}
