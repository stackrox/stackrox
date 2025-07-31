package manager

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
)

// BenchmarkUpdateComputerMemoryUsage compares memory usage between implementations
func BenchmarkUpdateComputerMemoryUsage(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Legacy_%d_connections", size), func(b *testing.B) {
			benchmarkLegacyMemory(b, size)
		})

		b.Run(fmt.Sprintf("Categorized_%d_connections", size), func(b *testing.B) {
			benchmarkCategorizedMemory(b, size)
		})
	}
}

func benchmarkLegacyMemory(b *testing.B, connectionCount int) {
	// Setup large dataset
	current, previous := generateConnectionMaps(connectionCount)
	legacy := NewLegacyUpdateComputer()
	// Set up legacy state
	legacy.UpdateState(previous, make(map[containerEndpointIndicator]timestamp.MicroTS), make(map[processListeningIndicator]timestamp.MicroTS))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = legacy.ComputeUpdatedConns(current)
	}
}

func benchmarkCategorizedMemory(b *testing.B, connectionCount int) {
	// Setup large dataset
	current, _ := generateConnectionMaps(connectionCount)

	mgr := &networkFlowManager{}
	categorized := NewCategorizedUpdateComputer(mgr)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = categorized.ComputeUpdatedConns(current)
	}
}

// BenchmarkUpdateComputerPerformance compares CPU performance between implementations
func BenchmarkUpdateComputerPerformance(b *testing.B) {
	scenarios := []struct {
		name             string
		connectionCount  int
		changePercentage float64 // Percentage of connections that are new/changed
	}{
		{"SmallDataset_HighChurn", 1000, 0.5},
		{"SmallDataset_LowChurn", 1000, 0.05},
		{"LargeDataset_HighChurn", 100000, 0.5},
		{"LargeDataset_LowChurn", 100000, 0.05},
	}

	for _, scenario := range scenarios {
		b.Run(fmt.Sprintf("Legacy_%s", scenario.name), func(b *testing.B) {
			benchmarkLegacyPerformance(b, scenario.connectionCount, scenario.changePercentage)
		})

		b.Run(fmt.Sprintf("Categorized_%s", scenario.name), func(b *testing.B) {
			benchmarkCategorizedPerformance(b, scenario.connectionCount, scenario.changePercentage)
		})
	}
}

func benchmarkLegacyPerformance(b *testing.B, connectionCount int, changePercentage float64) {
	current, previous := generateConnectionMapsWithChanges(connectionCount, changePercentage)
	legacy := NewLegacyUpdateComputer()
	// Set up legacy state
	legacy.UpdateState(previous, make(map[containerEndpointIndicator]timestamp.MicroTS), make(map[processListeningIndicator]timestamp.MicroTS))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = legacy.ComputeUpdatedConns(current)
	}
}

func benchmarkCategorizedPerformance(b *testing.B, connectionCount int, changePercentage float64) {
	current, _ := generateConnectionMapsWithChanges(connectionCount, changePercentage)

	mgr := &networkFlowManager{}
	categorized := NewCategorizedUpdateComputer(mgr)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = categorized.ComputeUpdatedConns(current)
	}
}

// BenchmarkStateTrackingMemory compares the memory overhead of state tracking
func BenchmarkStateTrackingMemory(b *testing.B) {
	sizes := []int{10000, 100000, 1000000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("LastSentState_%d", size), func(b *testing.B) {
			benchmarkLastSentStateMemory(b, size)
		})

		b.Run(fmt.Sprintf("FirstTimeSeen_%d", size), func(b *testing.B) {
			benchmarkFirstTimeSeenMemory(b, size)
		})
	}
}

func benchmarkLastSentStateMemory(b *testing.B, size int) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate LastSentState map
		state := make(map[networkConnIndicator]timestamp.MicroTS, size)
		for j := 0; j < size; j++ {
			conn := generateConnection(j)
			state[conn] = timestamp.Now()
		}
		_ = state
	}
}

func benchmarkFirstTimeSeenMemory(b *testing.B, size int) {
	b.ReportAllocs()

	mgr := &networkFlowManager{}
	categorized := NewCategorizedUpdateComputer(mgr).(*CategorizedUpdateComputer)

	for i := 0; i < b.N; i++ {
		// Simulate FirstTimeSeen set
		state := set.NewStringSet()
		for j := 0; j < size; j++ {
			conn := generateConnection(j)
			key := categorized.connectionKey(conn)
			state.Add(key)
		}
		_ = state
	}
}

// Helper functions for generating test data

func generateConnectionMaps(count int) (map[networkConnIndicator]timestamp.MicroTS, map[networkConnIndicator]timestamp.MicroTS) {
	current := make(map[networkConnIndicator]timestamp.MicroTS, count)
	previous := make(map[networkConnIndicator]timestamp.MicroTS, count)

	for i := 0; i < count; i++ {
		conn := generateConnection(i)
		current[conn] = timestamp.InfiniteFuture  // Open connections
		previous[conn] = timestamp.InfiniteFuture // Previously open
	}

	return current, previous
}

func generateConnectionMapsWithChanges(count int, changePercentage float64) (map[networkConnIndicator]timestamp.MicroTS, map[networkConnIndicator]timestamp.MicroTS) {
	current := make(map[networkConnIndicator]timestamp.MicroTS, count)
	previous := make(map[networkConnIndicator]timestamp.MicroTS, count)

	now := timestamp.Now()
	changeCount := int(float64(count) * changePercentage)

	// Add unchanged connections
	for i := 0; i < count-changeCount; i++ {
		conn := generateConnection(i)
		current[conn] = timestamp.InfiniteFuture
		previous[conn] = timestamp.InfiniteFuture
	}

	// Add changed connections (new in current)
	for i := count - changeCount; i < count; i++ {
		conn := generateConnection(i)
		current[conn] = now
		// Not in previous (new connection)
	}

	return current, previous
}

func generateConnection(id int) networkConnIndicator {
	srcEntity := networkgraph.Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   fmt.Sprintf("deployment-src-%d", id%100), // Reuse some entities
	}
	dstEntity := networkgraph.Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   fmt.Sprintf("deployment-dst-%d", (id+50)%100),
	}

	return networkConnIndicator{
		srcEntity: srcEntity,
		dstEntity: dstEntity,
		dstPort:   uint16(80 + (id % 1000)), // Vary ports
		protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
}

// BenchmarkMemoryFootprint measures the memory footprint of tracking structures
func BenchmarkMemoryFootprint(b *testing.B) {
	b.Run("LastSentStateMap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Measure memory of map[networkConnIndicator]timestamp.MicroTS
			m := make(map[networkConnIndicator]timestamp.MicroTS)
			conn := generateConnection(i)
			m[conn] = timestamp.Now()
			_ = m
		}
	})

	b.Run("StringSet", func(b *testing.B) {
		b.ReportAllocs()
		mgr := &networkFlowManager{}
		categorized := NewCategorizedUpdateComputer(mgr).(*CategorizedUpdateComputer)
		for i := 0; i < b.N; i++ {
			// Measure memory of set.StringSet (map[string]struct{})
			s := set.NewStringSet()
			conn := generateConnection(i)
			key := categorized.connectionKey(conn)
			s.Add(key)
			_ = s
		}
	})
}
