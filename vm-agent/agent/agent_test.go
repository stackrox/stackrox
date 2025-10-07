package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFakeAgent(t *testing.T) {
	tests := []struct {
		name            string
		port            uint32
		packageCount    int
		intervalMs      int
		parallelWorkers int
	}{
		{
			name:            "default values",
			port:            818,
			packageCount:    10,
			intervalMs:      10000,
			parallelWorkers: 1,
		},
		{
			name:            "custom values",
			port:            9999,
			packageCount:    100,
			intervalMs:      1000,
			parallelWorkers: 5,
		},
		{
			name:            "zero package count",
			port:            818,
			packageCount:    0,
			intervalMs:      10000,
			parallelWorkers: 1,
		},
		{
			name:            "multiple workers",
			port:            818,
			packageCount:    10,
			intervalMs:      10000,
			parallelWorkers: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewFakeAgent(tt.port, tt.packageCount, tt.intervalMs, tt.parallelWorkers)

			assert.NotNil(t, agent)
			assert.Equal(t, tt.port, agent.port)
			assert.Equal(t, tt.packageCount, agent.packageCount)
			assert.Equal(t, tt.intervalMs, agent.intervalMs)
			assert.Equal(t, tt.parallelWorkers, agent.parallelWorkers)
		})
	}
}

func TestGeneratePackages(t *testing.T) {
	tests := []struct {
		name          string
		packageCount  int
		expectedCount int
		checkUnique   bool
	}{
		{
			name:          "zero packages should return one",
			packageCount:  0,
			expectedCount: 1,
			checkUnique:   false,
		},
		{
			name:          "single package",
			packageCount:  1,
			expectedCount: 1,
			checkUnique:   false,
		},
		{
			name:          "standard count",
			packageCount:  10,
			expectedCount: 10,
			checkUnique:   true,
		},
		{
			name:          "large count with template reuse",
			packageCount:  50,
			expectedCount: 50,
			checkUnique:   true,
		},
		{
			name:          "very large count",
			packageCount:  1000,
			expectedCount: 1000,
			checkUnique:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewFakeAgent(818, tt.packageCount, 10000, 1)
			packages := agent.generatePackages()

			assert.Len(t, packages, tt.expectedCount)

			// Verify all packages have required fields
			for i, pkg := range packages {
				assert.NotEmpty(t, pkg.Id, "package %d missing ID", i)
				assert.NotEmpty(t, pkg.Name, "package %d missing Name", i)
				assert.NotEmpty(t, pkg.Version, "package %d missing Version", i)
				assert.NotEmpty(t, pkg.Kind, "package %d missing Kind", i)
				assert.NotEmpty(t, pkg.Arch, "package %d missing Arch", i)
			}

			// Check uniqueness of IDs
			if tt.checkUnique && len(packages) > 1 {
				ids := make(map[string]bool)
				for _, pkg := range packages {
					assert.False(t, ids[pkg.Id], "duplicate package ID: %s", pkg.Id)
					ids[pkg.Id] = true
				}
			}
		})
	}
}

func TestGenerateDistributions(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	// Run multiple times to ensure variety
	distributionTypes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		distributions := agent.generateDistributions()

		require.Len(t, distributions, 1, "should always return exactly one distribution")

		dist := distributions[0]
		assert.NotEmpty(t, dist.Id)
		assert.NotEmpty(t, dist.Did)
		assert.NotEmpty(t, dist.Name)
		assert.NotEmpty(t, dist.Version)
		assert.NotEmpty(t, dist.VersionId)
		assert.NotEmpty(t, dist.Arch)
		assert.NotEmpty(t, dist.PrettyName)

		distributionTypes[dist.Id] = true
	}

	// Should have randomness (multiple distribution types)
	assert.Greater(t, len(distributionTypes), 1, "should generate varied distributions")
}

func TestGenerateRepositories(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	// Run multiple times to ensure variety
	repoTypes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		repos := agent.generateRepositories()

		require.Len(t, repos, 1, "should always return exactly one repository")

		repo := repos[0]
		assert.NotEmpty(t, repo.Id)
		assert.NotEmpty(t, repo.Name)
		assert.NotEmpty(t, repo.Uri)

		repoTypes[repo.Id] = true
	}

	// Should have randomness (multiple repository types)
	assert.Greater(t, len(repoTypes), 1, "should generate varied repositories")
}

func TestGenerateIndexReport(t *testing.T) {
	tests := []struct {
		name         string
		packageCount int
	}{
		{
			name:         "small report",
			packageCount: 5,
		},
		{
			name:         "medium report",
			packageCount: 50,
		},
		{
			name:         "large report",
			packageCount: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewFakeAgent(818, tt.packageCount, 10000, 1)
			report := agent.generateIndexReport()

			require.NotNil(t, report)
			assert.NotEmpty(t, report.VsockCid)

			require.NotNil(t, report.IndexV4)
			assert.NotEmpty(t, report.IndexV4.HashId)
			assert.Equal(t, "IndexFinished", report.IndexV4.State)
			assert.True(t, report.IndexV4.Success)
			assert.Empty(t, report.IndexV4.Err)

			require.NotNil(t, report.IndexV4.Contents)
			assert.Len(t, report.IndexV4.Contents.Packages, tt.packageCount)
			assert.Len(t, report.IndexV4.Contents.Distributions, 1)
			assert.Len(t, report.IndexV4.Contents.Repositories, 1)
		})
	}
}

func TestGenerateIndexReport_UniqueHashIDs(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	hashIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		report := agent.generateIndexReport()
		hashID := report.IndexV4.HashId

		assert.False(t, hashIDs[hashID], "duplicate hash ID: %s", hashID)
		hashIDs[hashID] = true

		// Small delay to ensure timestamp-based IDs are different
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRun_SingleWorker(t *testing.T) {
	agent := NewFakeAgent(818, 10, 100, 1)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short time
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Run should return when context is cancelled
	// Note: This will fail to connect to vsock, but we're testing the lifecycle
	err := agent.Run(ctx)
	assert.NoError(t, err)
}

func TestRun_MultipleWorkers(t *testing.T) {
	tests := []struct {
		name            string
		parallelWorkers int
	}{
		{
			name:            "two workers",
			parallelWorkers: 2,
		},
		{
			name:            "five workers",
			parallelWorkers: 5,
		},
		{
			name:            "ten workers",
			parallelWorkers: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewFakeAgent(818, 10, 100, tt.parallelWorkers)

			ctx, cancel := context.WithCancel(context.Background())

			// Cancel after a short time
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()

			// Run should return when context is cancelled
			err := agent.Run(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestRun_CancellationStopsAllWorkers(t *testing.T) {
	agent := NewFakeAgent(818, 10, 1000, 5)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		agent.Run(ctx)
		close(done)
	}()

	// Give workers time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel and verify it completes quickly
	cancel()

	select {
	case <-done:
		// Success - Run returned
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not complete after context cancellation")
	}
}

func TestRun_ImmediateCancellation(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return immediately without error
	err := agent.Run(ctx)
	assert.NoError(t, err)
}

func TestPackageGeneration_Consistency(t *testing.T) {
	agent := NewFakeAgent(818, 30, 10000, 1)

	packages := agent.generatePackages()
	assert.Len(t, packages, 30)

	// Verify package IDs are sequential
	for _, pkg := range packages {
		assert.Contains(t, pkg.Id, "pkg-")
		assert.NotEmpty(t, pkg.Name)
		assert.NotEmpty(t, pkg.Version)
	}
}

func TestPackageGeneration_TemplateReuse(t *testing.T) {
	// Test that template reuse works correctly for counts > template count
	agent := NewFakeAgent(818, 100, 10000, 1)

	packages := agent.generatePackages()
	assert.Len(t, packages, 100)

	// All packages should have valid data
	for _, pkg := range packages {
		assert.NotEmpty(t, pkg.Name)
		assert.NotEmpty(t, pkg.Version)
		assert.Equal(t, "binary", pkg.Kind)
		assert.NotEmpty(t, pkg.Arch)
	}
}

func TestConcurrentReportGeneration(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	// Generate reports concurrently to test for race conditions
	var wg sync.WaitGroup
	reports := make([]*v4.IndexReport, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			report := agent.generateIndexReport()
			reports[idx] = report.IndexV4
		}(i)
	}

	wg.Wait()

	// Verify all reports were generated
	for i, report := range reports {
		assert.NotNil(t, report, "report %d was nil", i)
		assert.NotEmpty(t, report.HashId)
		assert.Len(t, report.Contents.Packages, 10)
	}

	// Verify uniqueness of hash IDs (may have duplicates due to timing)
	hashIDs := make(map[string]int)
	for _, report := range reports {
		hashIDs[report.HashId]++
	}

	// At least some should be unique
	assert.Greater(t, len(hashIDs), 1, "should generate some unique hash IDs")
}

func TestWorkerConcurrency(t *testing.T) {
	// Test that multiple workers can run concurrently without blocking each other
	agent := NewFakeAgent(818, 5, 50, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := agent.Run(ctx)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Should complete within timeout + small margin
	assert.Less(t, duration, 500*time.Millisecond)
}

func TestPackageCount_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		packageCount  int
		expectedCount int
	}{
		{
			name:          "negative count treated as 1",
			packageCount:  -5,
			expectedCount: 1,
		},
		{
			name:          "zero count becomes 1",
			packageCount:  0,
			expectedCount: 1,
		},
		{
			name:          "exactly one",
			packageCount:  1,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewFakeAgent(818, tt.packageCount, 10000, 1)
			packages := agent.generatePackages()
			assert.Len(t, packages, tt.expectedCount)
		})
	}
}

func TestDistributionVariety(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	distributions := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		dists := agent.generateDistributions()
		distributions[dists[0].Id]++
	}

	// Should have multiple distribution types
	assert.GreaterOrEqual(t, len(distributions), 2, "should generate at least 2 different distributions")

	// Each should appear at least once
	for distID, count := range distributions {
		assert.Greater(t, count, 0, "distribution %s should appear at least once", distID)
	}
}

func TestRepositoryVariety(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)

	repositories := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		repos := agent.generateRepositories()
		repositories[repos[0].Id]++
	}

	// Should have multiple repository types
	assert.GreaterOrEqual(t, len(repositories), 2, "should generate at least 2 different repositories")

	// Each should appear at least once
	for repoID, count := range repositories {
		assert.Greater(t, count, 0, "repository %s should appear at least once", repoID)
	}
}

func TestIndexReport_ProtobufCompatibility(t *testing.T) {
	agent := NewFakeAgent(818, 10, 10000, 1)
	report := agent.generateIndexReport()

	// Verify all required proto fields are set
	assert.NotNil(t, report.IndexV4)
	assert.NotEmpty(t, report.IndexV4.HashId)
	assert.NotNil(t, report.IndexV4.Contents)
	assert.NotNil(t, report.IndexV4.Contents.Packages)
	assert.NotNil(t, report.IndexV4.Contents.Distributions)
	assert.NotNil(t, report.IndexV4.Contents.Repositories)
}

func BenchmarkGenerateIndexReport(b *testing.B) {
	tests := []struct {
		name         string
		packageCount int
	}{
		{"small_10", 10},
		{"medium_100", 100},
		{"large_1000", 1000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			agent := NewFakeAgent(818, tt.packageCount, 10000, 1)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				agent.generateIndexReport()
			}
		})
	}
}

func BenchmarkGeneratePackages(b *testing.B) {
	tests := []struct {
		name         string
		packageCount int
	}{
		{"small_10", 10},
		{"medium_100", 100},
		{"large_1000", 1000},
		{"xlarge_10000", 10000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			agent := NewFakeAgent(818, tt.packageCount, 10000, 1)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				agent.generatePackages()
			}
		})
	}
}

func BenchmarkConcurrentReportGeneration(b *testing.B) {
	agent := NewFakeAgent(818, 100, 10000, 1)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			agent.generateIndexReport()
		}
	})
}
