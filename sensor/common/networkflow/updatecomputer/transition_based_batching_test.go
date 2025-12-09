package updatecomputer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// Test data setup
	entity1 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}
	entity3 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-3"}
	entity4 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-4"}
	entity5 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-5"}
	entity6 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-6"}
	entity7 = networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-7"}

	conn12 = indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   8012,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn23 = indicator.NetworkConn{
		SrcEntity: entity2,
		DstEntity: entity3,
		DstPort:   8023,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn34 = indicator.NetworkConn{
		SrcEntity: entity3,
		DstEntity: entity4,
		DstPort:   8034,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn45 = indicator.NetworkConn{
		SrcEntity: entity4,
		DstEntity: entity5,
		DstPort:   8045,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn56 = indicator.NetworkConn{
		SrcEntity: entity5,
		DstEntity: entity6,
		DstPort:   8056,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn67 = indicator.NetworkConn{
		SrcEntity: entity6,
		DstEntity: entity7,
		DstPort:   8067,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	ep1 = indicator.ContainerEndpoint{
		Entity:   entity1,
		Port:     8080,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	ep2 = indicator.ContainerEndpoint{
		Entity:   entity2,
		Port:     8081,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	ep3 = indicator.ContainerEndpoint{
		Entity:   entity3,
		Port:     8082,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	ep4 = indicator.ContainerEndpoint{
		Entity:   entity4,
		Port:     8083,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	ep5 = indicator.ContainerEndpoint{
		Entity:   entity5,
		Port:     8084,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	ep6 = indicator.ContainerEndpoint{
		Entity:   entity6,
		Port:     8085,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	proc1 = indicator.ProcessListening{
		DeploymentID:  "deployment-1",
		PodID:         "pod-1",
		ContainerName: "container-1",
		PodUID:        "uid-1",
		Namespace:     "namespace-1",
		Port:          8080,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-1",
			ProcessArgs: "args-1",
			ProcessExec: "exec-1",
		},
	}
	proc2 = indicator.ProcessListening{
		DeploymentID:  "deployment-2",
		PodID:         "pod-2",
		ContainerName: "container-2",
		PodUID:        "uid-2",
		Namespace:     "namespace-2",
		Port:          8081,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-2",
			ProcessArgs: "args-2",
			ProcessExec: "exec-2",
		},
	}
	proc3 = indicator.ProcessListening{
		DeploymentID:  "deployment-3",
		PodID:         "pod-3",
		ContainerName: "container-3",
		PodUID:        "uid-3",
		Namespace:     "namespace-3",
		Port:          8082,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-3",
			ProcessArgs: "args-3",
			ProcessExec: "exec-3",
		},
	}
	proc4 = indicator.ProcessListening{
		DeploymentID:  "deployment-4",
		PodID:         "pod-4",
		ContainerName: "container-4",
		PodUID:        "uid-4",
		Namespace:     "namespace-4",
		Port:          8083,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-4",
			ProcessArgs: "args-4",
			ProcessExec: "exec-4",
		},
	}
	proc5 = indicator.ProcessListening{
		DeploymentID:  "deployment-5",
		PodID:         "pod-5",
		ContainerName: "container-5",
		PodUID:        "uid-5",
		Namespace:     "namespace-5",
		Port:          8084,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-5",
			ProcessArgs: "args-5",
			ProcessExec: "exec-5",
		},
	}
	proc6 = indicator.ProcessListening{
		DeploymentID:  "deployment-6",
		PodID:         "pod-6",
		ContainerName: "container-6",
		PodUID:        "uid-6",
		Namespace:     "namespace-6",
		Port:          8085,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		Process: indicator.ProcessInfo{
			ProcessName: "process-6",
			ProcessArgs: "args-6",
			ProcessExec: "exec-6",
		},
	}

	emptyConnUpdate = map[indicator.NetworkConn]timestamp.MicroTS{}
	emptyProcUpdate = map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{}

	open   = timestamp.InfiniteFuture
	closed = timestamp.Now()
)

// TestTransitionBasedConnectionBatching tests the connection batching behavior.
func TestTransitionBasedConnectionBatching(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")

	t.Run("batching returns at most maxUpdateSize flows from cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 4 new connections (all will be cached after compute)
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
			conn23: open,
			conn34: open,
			conn45: open,
		}

		flows := uc.ComputeUpdatedConns(update1)
		// With batching enabled, should return only 3 flows (max batch size)
		assert.Len(t, flows, 3)

		// Call successful send (with batching, cache should NOT be cleared)
		uc.OnSuccessfulSendConnections(update1)

		// Next call with empty update should return remaining 1 flow
		flows = uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows, 1)

		// Call successful send again
		uc.OnSuccessfulSendConnections(emptyConnUpdate)

		// Next call should return empty
		flows = uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows, 0)
	})

	t.Run("batching allows cache to grow when less than maxUpdateSize", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 2 new connections (less than max batch size of 3)
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
			conn23: open,
		}

		flows := uc.ComputeUpdatedConns(update1)
		// Should return all 2 flows since it's less than batch size
		assert.Len(t, flows, 2)

		// Cache should be empty after successful send
		uc.OnSuccessfulSendConnections(update1)
		flows = uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows, 0)
	})
}

// TestTransitionBasedConnectionFailureHandling tests the OnSendConnectionsFailure behavior.
func TestTransitionBasedConnectionFailureHandling(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")

	t.Run("failure handler re-adds unsent flows to front of cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 3 new connections
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
			conn23: open,
			conn34: open,
		}

		flows := uc.ComputeUpdatedConns(update1)
		require.Len(t, flows, 3)

		// Simulate send failure by calling OnSendConnectionsFailure
		uc.OnSendConnectionsFailure(flows)

		// Next call should return the same flows again (from front of cache)
		flows2 := uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows2, 3)

		// Verify the flows are the same (order might differ, but all should be present)
		protoassert.SlicesEqual(t, flows, flows2)
	})

	t.Run("failure handler preserves cache ordering", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add connections one by one with failures
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
		}
		update2 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn23: open,
		}
		update3 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn34: open,
		}

		// Process first update and simulate failure
		flows1 := uc.ComputeUpdatedConns(update1)
		require.Len(t, flows1, 1)           // Returns conn12
		uc.OnSendConnectionsFailure(flows1) // Prepend conn12 back to cache

		// Process second update - should return both conn12 (from cache) and conn23 (new)
		flows2 := uc.ComputeUpdatedConns(update2)
		require.Len(t, flows2, 2)           // Returns [conn12, conn23]
		uc.OnSendConnectionsFailure(flows2) // Prepend both back to cache

		// Process third update - should return all three
		flows3 := uc.ComputeUpdatedConns(update3)
		require.Len(t, flows3, 3) // Returns [conn12, conn23, conn34]

		// Verify all 3 connections are present
		protoassert.SlicesEqual(t, flows3, []*storage.NetworkFlow{
			conn12.ToProto(open),
			conn23.ToProto(open),
			conn34.ToProto(open),
		})
	})
}

// TestTransitionBasedCacheLimiting tests the cache limiting behavior.
func TestTransitionBasedCacheLimiting(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "false")
	t.Setenv("ROX_NETFLOW_CACHE_LIMITING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")

	t.Run("cache limiting discards open flows when exceeding maxCacheSize", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 6 open connections (exceeds cache size of 5)
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
			conn23: open,
			conn34: open,
			conn45: open,
			conn56: open,
			conn67: open,
		}

		flows := uc.ComputeUpdatedConns(update1)
		// Cache limiting applies immediately, so only 5 are returned (1 open flow discarded)
		assert.Len(t, flows, 5)

		uc.OnSuccessfulSendConnections(update1)

		// Next call with empty update should return 0 (cache was cleared)
		flows = uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows, 0)
	})

	t.Run("cache limiting prioritizes closed flows over open flows", func(t *testing.T) {
		uc := NewTransitionBased()

		// First, establish some open connections
		initialUpdate := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: open,
			conn23: open,
			conn34: open,
		}
		flows := uc.ComputeUpdatedConns(initialUpdate)
		uc.OnSuccessfulSendConnections(initialUpdate)
		assert.Len(t, flows, 3)

		// Now add 3 closed and 3 open connections (total 6, exceeds cache of 5)
		update1 := map[indicator.NetworkConn]timestamp.MicroTS{
			conn12: closed, // Close previously open
			conn23: closed, // Close previously open
			conn34: closed, // Close previously open
			conn45: open,   // New open
			conn56: open,   // New open
			conn67: open,   // New open
		}

		flows = uc.ComputeUpdatedConns(update1)
		// Cache limiting applies immediately, keeping 3 closed + 2 open (discarding 1 open)
		assert.Len(t, flows, 5)

		// Verify that the closed connections are in the result
		var closedCount int
		for _, flow := range flows {
			if isConnClosed(flow) {
				closedCount++
			}
		}
		assert.Equal(t, 3, closedCount, "Cache should prioritize closed connections - all 3 should be present")

		uc.OnSuccessfulSendConnections(update1)
		flows = uc.ComputeUpdatedConns(emptyConnUpdate)
		assert.Len(t, flows, 0)
	})
}

// TestTransitionBasedEndpointBatching tests the endpoint batching behavior.
func TestTransitionBasedEndpointBatching(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")

	t.Run("batching returns at most maxUpdateSize endpoints from cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 4 new endpoints (all will be cached after compute)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: nil},
			ep2: {LastSeen: open, ProcessListening: nil},
			ep3: {LastSeen: open, ProcessListening: nil},
			ep4: {LastSeen: open, ProcessListening: nil},
		}

		eps, _ := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// With batching enabled, should return only 3 endpoints (max batch size)
		assert.Len(t, eps, 3)

		// Call successful send
		uc.OnSuccessfulSendEndpoints(update1)

		// Next call with empty update should return remaining 1 endpoint
		eps, _ = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, eps, 1)

		// Call successful send again
		uc.OnSuccessfulSendEndpoints(emptyProcUpdate)

		// Next call should return empty
		eps, _ = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, eps, 0)
	})

	t.Run("endpoint failure handler re-adds unsent endpoints to front of cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 3 new endpoints
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: nil},
			ep2: {LastSeen: open, ProcessListening: nil},
			ep3: {LastSeen: open, ProcessListening: nil},
		}

		eps, _ := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		require.Len(t, eps, 3)

		// Simulate send failure by calling OnSendEndpointsFailure
		uc.OnSendEndpointsFailure(eps)

		// Next call should return the same endpoints again (from front of cache)
		eps2, _ := uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, eps2, 3)

		// Verify the endpoints are the same (order might differ, but all should be present)
		protoassert.SlicesEqual(t, eps, eps2)
	})
}

// TestTransitionBasedEndpointCacheLimiting tests the endpoint cache limiting behavior.
func TestTransitionBasedEndpointCacheLimiting(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "false")
	t.Setenv("ROX_NETFLOW_CACHE_LIMITING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")

	t.Run("cache limiting discards open endpoints when exceeding maxCacheSize", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 6 open endpoints (exceeds cache size of 5)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: nil},
			ep2: {LastSeen: open, ProcessListening: nil},
			ep3: {LastSeen: open, ProcessListening: nil},
			ep4: {LastSeen: open, ProcessListening: nil},
			ep5: {LastSeen: open, ProcessListening: nil},
			ep6: {LastSeen: open, ProcessListening: nil},
		}

		eps, _ := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// Cache limiting applies immediately, so only 5 are returned (1 open endpoint discarded)
		assert.Len(t, eps, 5)

		uc.OnSuccessfulSendEndpoints(update1)

		// Next call with empty update should return 0 (cache was cleared)
		eps, _ = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, eps, 0)
	})

	t.Run("cache limiting prioritizes closed endpoints over open endpoints", func(t *testing.T) {
		uc := NewTransitionBased()

		// First, establish some open endpoints
		initialUpdate := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: nil},
			ep2: {LastSeen: open, ProcessListening: nil},
			ep3: {LastSeen: open, ProcessListening: nil},
		}
		eps, _ := uc.ComputeUpdatedEndpointsAndProcesses(initialUpdate)
		uc.OnSuccessfulSendEndpoints(initialUpdate)
		assert.Len(t, eps, 3)

		// Now add 3 closed and 3 open endpoints (total 6, exceeds cache of 5)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: closed, ProcessListening: nil}, // Close previously open
			ep2: {LastSeen: closed, ProcessListening: nil}, // Close previously open
			ep3: {LastSeen: closed, ProcessListening: nil}, // Close previously open
			ep4: {LastSeen: open, ProcessListening: nil},   // New open
			ep5: {LastSeen: open, ProcessListening: nil},   // New open
			ep6: {LastSeen: open, ProcessListening: nil},   // New open
		}

		eps, _ = uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// Cache limiting applies immediately, keeping 3 closed + 2 open (discarding 1 open)
		assert.Len(t, eps, 5)

		// Verify that the closed endpoints are in the result
		var closedCount int
		for _, ep := range eps {
			if isEndpointClosed(ep) {
				closedCount++
			}
		}
		assert.Equal(t, 3, closedCount, "Cache should prioritize closed endpoints - all 3 should be present")

		// Clear cache to verify
		uc.OnSuccessfulSendEndpoints(update1)
		eps, _ = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, eps, 0)
	})
}

// TestLimitCacheSize tests the generic limitCacheSize function
func TestLimitCacheSize(t *testing.T) {

	t.Run("does not modify cache when under limit", func(t *testing.T) {
		cache := []*storage.NetworkFlow{
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now())},
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now())},
		}
		result, dropped := limitCacheSize(cache, 5, isConnClosed)
		assert.Len(t, result, 2)
		assert.Equal(t, 0, dropped)
	})

	t.Run("limits cache to maxSize and prioritizes closed items", func(t *testing.T) {
		closed := timestamp.Now()

		cache := []*storage.NetworkFlow{
			{LastSeenTimestamp: nil}, // open - should be discarded
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)}, // closed - should be kept
			{LastSeenTimestamp: nil}, // open - should be discarded
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)}, // closed - should be kept
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)}, // closed - should be kept
		}

		result, dropped := limitCacheSize(cache, 3, isConnClosed)
		assert.Len(t, result, 3)
		assert.Equal(t, 2, dropped)

		// All kept items should be closed
		for _, flow := range result {
			assert.True(t, isConnClosed(flow), "Expected all kept items to be closed")
		}
	})

	t.Run("discards open items first when limit is exceeded", func(t *testing.T) {
		cache := []*storage.NetworkFlow{
			{LastSeenTimestamp: nil}, // open - should be discarded
			{LastSeenTimestamp: nil}, // open - should be discarded
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)}, // closed - should be kept
		}

		result, dropped := limitCacheSize(cache, 1, isConnClosed)
		assert.Len(t, result, 1)
		assert.Equal(t, 2, dropped)
		assert.True(t, isConnClosed(result[0]), "Expected the kept item to be closed")
	})

	t.Run("handles all closed items correctly", func(t *testing.T) {
		cache := []*storage.NetworkFlow{
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)},
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)},
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)},
			{LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(closed)},
		}

		result, dropped := limitCacheSize(cache, 2, isConnClosed)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, dropped)
	})
}

// TestIsConnClosed tests the isConnClosed helper function
func TestIsConnClosed(t *testing.T) {
	t.Run("returns true for closed connection", func(t *testing.T) {
		flow := &storage.NetworkFlow{
			LastSeenTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
		}
		assert.True(t, isConnClosed(flow))
	})

	t.Run("returns false for open connection (nil timestamp)", func(t *testing.T) {
		flow := &storage.NetworkFlow{
			LastSeenTimestamp: nil,
		}
		assert.False(t, isConnClosed(flow))
	})

	t.Run("returns false for nil timestamp", func(t *testing.T) {
		flow := &storage.NetworkFlow{
			LastSeenTimestamp: nil,
		}
		assert.False(t, isConnClosed(flow))
	})
}

// TestIsEndpointClosed tests the isEndpointClosed helper function
func TestIsEndpointClosed(t *testing.T) {
	t.Run("returns true for closed endpoint", func(t *testing.T) {
		ep := &storage.NetworkEndpoint{
			LastActiveTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
		}
		assert.True(t, isEndpointClosed(ep))
	})

	t.Run("returns false for open endpoint (nil timestamp)", func(t *testing.T) {
		ep := &storage.NetworkEndpoint{
			LastActiveTimestamp: nil,
		}
		assert.False(t, isEndpointClosed(ep))
	})

	t.Run("returns false for nil timestamp", func(t *testing.T) {
		ep := &storage.NetworkEndpoint{
			LastActiveTimestamp: nil,
		}
		assert.False(t, isEndpointClosed(ep))
	})
}

// TestIsProcClosed tests the isProcClosed helper function
func TestIsProcClosed(t *testing.T) {
	t.Run("returns true for closed process", func(t *testing.T) {
		proc := &storage.ProcessListeningOnPortFromSensor{
			CloseTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
		}
		assert.True(t, isProcClosed(proc))
	})

	t.Run("returns false for open process (nil timestamp)", func(t *testing.T) {
		proc := &storage.ProcessListeningOnPortFromSensor{
			CloseTimestamp: nil,
		}
		assert.False(t, isProcClosed(proc))
	})

	t.Run("returns false for open process (infinite future timestamp)", func(t *testing.T) {
		proc := &storage.ProcessListeningOnPortFromSensor{
			CloseTimestamp: protoconv.ConvertMicroTSToProtobufTS(timestamp.InfiniteFuture),
		}
		assert.False(t, isProcClosed(proc))
	})
}

// TestTransitionBasedProcessBatching tests the process batching behavior.
func TestTransitionBasedProcessBatching(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")
	t.Setenv("ROX_PROCESSES_LISTENING_ON_PORT", "true")

	t.Run("batching returns at most maxUpdateSize processes from cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 4 new processes (all will be cached after compute)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
			ep4: {LastSeen: open, ProcessListening: &proc4},
		}

		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// With batching enabled, should return only 3 processes (max batch size)
		assert.Len(t, procs, 3)

		// Call successful send
		uc.OnSuccessfulSendProcesses(update1)

		// Next call with empty update should return remaining 1 process
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 1)

		// Call successful send again
		uc.OnSuccessfulSendProcesses(emptyProcUpdate)

		// Next call should return empty
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 0)
	})

	t.Run("batching allows cache to grow when less than maxUpdateSize", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 2 new processes (less than max batch size of 3)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
		}

		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// Should return all 2 processes since it's less than batch size
		assert.Len(t, procs, 2)

		// Cache should be empty after successful send
		uc.OnSuccessfulSendProcesses(update1)
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 0)
	})

	t.Run("process failure handler re-adds unsent processes to front of cache", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 3 new processes
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
		}

		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		require.Len(t, procs, 3)

		// Simulate send failure by calling OnSendProcessesFailure
		uc.OnSendProcessesFailure(procs)

		// Next call should return the same processes again (from front of cache)
		_, procs2 := uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs2, 3)

		// Verify the processes are the same (order might differ, but all should be present)
		protoassert.SlicesEqual(t, procs, procs2)
	})

	t.Run("process failure handler preserves cache ordering", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add processes one by one with failures
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
		}
		update2 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep2: {LastSeen: open, ProcessListening: &proc2},
		}
		update3 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep3: {LastSeen: open, ProcessListening: &proc3},
		}

		// Process first update and simulate failure
		_, procs1 := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		require.Len(t, procs1, 1)         // Returns proc1
		uc.OnSendProcessesFailure(procs1) // Prepend proc1 back to cache

		// Process second update - should return both proc1 (from cache) and proc2 (new)
		_, procs2 := uc.ComputeUpdatedEndpointsAndProcesses(update2)
		require.Len(t, procs2, 2)         // Returns [proc1, proc2]
		uc.OnSendProcessesFailure(procs2) // Prepend both back to cache

		// Process third update - should return all three
		_, procs3 := uc.ComputeUpdatedEndpointsAndProcesses(update3)
		require.Len(t, procs3, 3) // Returns [proc1, proc2, proc3]

		// Verify all 3 processes are present
		protoassert.SlicesEqual(t, procs3, []*storage.ProcessListeningOnPortFromSensor{
			proc1.ToProto(open),
			proc2.ToProto(open),
			proc3.ToProto(open),
		})
	})
}

// TestTransitionBasedProcessCacheLimiting tests the process cache limiting behavior when PLOP is enabled.
func TestTransitionBasedProcessCacheLimiting(t *testing.T) {
	t.Setenv("ROX_NETFLOW_BATCHING", "false")
	t.Setenv("ROX_NETFLOW_CACHE_LIMITING", "true")
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")
	t.Setenv("ROX_PROCESSES_LISTENING_ON_PORT", "true")

	t.Run("cache limiting discards open processes when exceeding maxCacheSize", func(t *testing.T) {
		uc := NewTransitionBased()

		// Add 6 open processes (exceeds cache size of 5)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
			ep4: {LastSeen: open, ProcessListening: &proc4},
			ep5: {LastSeen: open, ProcessListening: &proc5},
			ep6: {LastSeen: open, ProcessListening: &proc6},
		}

		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// Cache limiting applies immediately, so only 5 are returned (1 open process discarded)
		assert.Len(t, procs, 5)

		uc.OnSuccessfulSendProcesses(update1)

		// Next call with empty update should return 0 (cache was cleared)
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 0)
	})

	t.Run("cache limiting prioritizes closed processes over open processes", func(t *testing.T) {
		uc := NewTransitionBased()

		// First, establish some open processes
		initialUpdate := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
		}
		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(initialUpdate)
		uc.OnSuccessfulSendProcesses(initialUpdate)
		assert.Len(t, procs, 3)

		// Now add 3 closed and 3 open processes (total 6, exceeds cache of 5)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: closed, ProcessListening: &proc1}, // Close previously open
			ep2: {LastSeen: closed, ProcessListening: &proc2}, // Close previously open
			ep3: {LastSeen: closed, ProcessListening: &proc3}, // Close previously open
			ep4: {LastSeen: open, ProcessListening: &proc4},   // New open
			ep5: {LastSeen: open, ProcessListening: &proc5},   // New open
			ep6: {LastSeen: open, ProcessListening: &proc6},   // New open
		}

		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(update1)
		// Cache limiting applies immediately, keeping 3 closed + 2 open (discarding 1 open)
		assert.Len(t, procs, 5)

		// Verify that the closed processes are in the result
		var closedCount int
		for _, proc := range procs {
			if isProcClosed(proc) {
				closedCount++
			}
		}
		assert.Equal(t, 3, closedCount, "Cache should prioritize closed processes - all 3 should be present")

		// Clear cache to verify
		uc.OnSuccessfulSendProcesses(update1)
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 0)
	})
}

// TestOnSuccessfulSendProcesses tests the OnSuccessfulSendProcesses behavior with batching enabled/disabled.
func TestOnSuccessfulSendProcesses(t *testing.T) {
	t.Setenv("ROX_NETFLOW_MAX_UPDATE_SIZE", "3")
	t.Setenv("ROX_NETFLOW_MAX_CACHE_SIZE", "5")
	t.Setenv("ROX_PROCESSES_LISTENING_ON_PORT", "true")

	t.Run("with batching disabled, OnSuccessfulSendProcesses clears process cache", func(t *testing.T) {
		t.Setenv("ROX_NETFLOW_BATCHING", "false")
		uc := NewTransitionBased()

		// Add 4 processes
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
			ep4: {LastSeen: open, ProcessListening: &proc4},
		}

		// With batching disabled, all 4 processes should be returned
		_, procs := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		require.Len(t, procs, 4)

		// Call successful send
		uc.OnSuccessfulSendProcesses(update1)

		// Next call with empty update should return 0 (cache was cleared)
		_, procs = uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs, 0)
	})

	t.Run("with batching enabled, OnSuccessfulSendProcesses does NOT clear process cache", func(t *testing.T) {
		t.Setenv("ROX_NETFLOW_BATCHING", "true")
		uc := NewTransitionBased()

		// Add 4 processes (more than max batch size of 3)
		update1 := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp{
			ep1: {LastSeen: open, ProcessListening: &proc1},
			ep2: {LastSeen: open, ProcessListening: &proc2},
			ep3: {LastSeen: open, ProcessListening: &proc3},
			ep4: {LastSeen: open, ProcessListening: &proc4},
		}

		// First call returns 3 processes (batch size)
		_, procs1 := uc.ComputeUpdatedEndpointsAndProcesses(update1)
		require.Len(t, procs1, 3)

		// Call successful send - with batching enabled, cache should NOT be cleared
		uc.OnSuccessfulSendProcesses(update1)

		// Next call should return remaining 1 process from cache
		_, procs2 := uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs2, 1, "With batching enabled, OnSuccessfulSendProcesses should NOT clear the cache")

		// Call successful send again
		uc.OnSuccessfulSendProcesses(emptyProcUpdate)

		// Now cache should be empty
		_, procs3 := uc.ComputeUpdatedEndpointsAndProcesses(emptyProcUpdate)
		assert.Len(t, procs3, 0)
	})
}
