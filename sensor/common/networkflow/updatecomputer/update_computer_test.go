package updatecomputer

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
)

// TODO: For me: Those tests are messy and there are many duplications.
// Rewrite them with NI carefully and make sure they are correct.

func TestUpdateComputerImplementations(t *testing.T) {
	// Test data setup
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

	conn1 := &indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000
	future := now + 1000

	testCases := map[string]struct {
		current                map[*indicator.NetworkConn]timestamp.MicroTS
		previous               map[*indicator.NetworkConn]timestamp.MicroTS
		expectCountLegacy      int
		expectCountCategorized int
	}{
		"should send new connections": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: now,
			},
			previous:               map[*indicator.NetworkConn]timestamp.MicroTS{},
			expectCountLegacy:      1,
			expectCountCategorized: 1,
		},
		"should send when connection closes": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: past, // closed connection
			},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // was open
			},
			expectCountLegacy:      1,
			expectCountCategorized: 1,
		},
		"should not send duplicate open connections": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			expectCountLegacy:      0, // Legacy: same timestamps = no update
			expectCountCategorized: 0, // Categorized: same timestamps = skip
		},
		"open connection removal legacy reports categorized does not": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // was open
			},
			expectCountLegacy:      1, // Legacy tracks all removals
			expectCountCategorized: 0, // Categorized doesn't track open connection removals
		},
		"closed connection removal both should report": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: past, // was closed
			},
			expectCountLegacy:      1, // Legacy tracks all removals
			expectCountCategorized: 1, // Categorized should track closed connection removals (BUG?)
		},
		"should send when timestamp is newer": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: future,
			},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: now,
			},
			expectCountLegacy:      1,
			expectCountCategorized: 1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Test both implementations
			t.Run("Legacy", func(t *testing.T) {
				legacy := NewLegacy()
				// For legacy implementation, we need to first set up the state
				legacy.UpdateState(tc.previous, make(map[*indicator.ContainerEndpoint]timestamp.MicroTS), make(map[*indicator.ProcessListening]timestamp.MicroTS))

				results := legacy.ComputeUpdatedConns(tc.current)
				assert.Len(t, results, tc.expectCountLegacy, "Legacy: %s", name)
			})

			t.Run("Categorized", func(t *testing.T) {
				categorized := NewCategorized()
				results := categorized.ComputeUpdatedConns(tc.current)
				assert.Len(t, results, tc.expectCountCategorized, "Categorized: %s", name)
			})
		})
	}
}

func TestUpdateComputerBehavioralEquivalence(t *testing.T) {
	// Test that both implementations produce the same results for critical scenarios
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

	conn1 := &indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000

	// Scenarios for testing behavioral equivalence
	equivalentCases := map[string]struct {
		current                 map[*indicator.NetworkConn]timestamp.MicroTS
		previous                map[*indicator.NetworkConn]timestamp.MicroTS
		expectEquivalentResults bool
	}{
		"new connection should be sent by both": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: now,
			},
			previous:                map[*indicator.NetworkConn]timestamp.MicroTS{},
			expectEquivalentResults: true,
		},
		"closed connection should be sent by both": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: past,
			},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			expectEquivalentResults: true,
		},
		"open connection removed different behavior": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // was open
			},
			expectEquivalentResults: false, // Categorized doesn't track open connection removals
		},
		"closed connection removed should be equivalent": {
			current: map[*indicator.NetworkConn]timestamp.MicroTS{},
			previous: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: past, // was closed
			},
			expectEquivalentResults: true, // Both should track closed connection removals
		},
	}

	for name, tc := range equivalentCases {
		t.Run(name, func(t *testing.T) {
			legacy := NewLegacy()
			// Set up legacy state
			legacy.UpdateState(tc.previous, make(map[*indicator.ContainerEndpoint]timestamp.MicroTS), make(map[*indicator.ProcessListening]timestamp.MicroTS))

			categorized := NewCategorized()

			legacyResults := legacy.ComputeUpdatedConns(tc.current)
			categorizedResults := categorized.ComputeUpdatedConns(tc.current)

			if tc.expectEquivalentResults {
				assert.Equal(t, len(legacyResults), len(categorizedResults),
					"Both implementations should produce same number of updates for: %s", name)
			} else {
				// Different behavior expected - just log the difference
				t.Logf("Different behavior case: Legacy=%d, Categorized=%d (as expected)",
					len(legacyResults), len(categorizedResults))
			}

			// Should produce semantically equivalent results
			if len(legacyResults) > 0 && len(categorizedResults) > 0 {
				// Check that the connections are the same
				assert.Equal(t, legacyResults[0].Props.SrcEntity, categorizedResults[0].Props.SrcEntity)
				assert.Equal(t, legacyResults[0].Props.DstEntity, categorizedResults[0].Props.DstEntity)
				assert.Equal(t, legacyResults[0].Props.DstPort, categorizedResults[0].Props.DstPort)
			}
		})
	}
}

// TestComputeUpdatedEndpoints tests endpoint update computation for both implementations
func TestComputeUpdatedEndpoints(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}

	endpoint1 := &indicator.ContainerEndpoint{
		Entity:   entity1,
		Port:     8080,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000

	testCases := map[string]struct {
		current     map[*indicator.ContainerEndpoint]timestamp.MicroTS
		expectCount int
		description string
	}{
		"new endpoint": {
			current: map[*indicator.ContainerEndpoint]timestamp.MicroTS{
				endpoint1: now,
			},
			expectCount: 1,
			description: "Should send new endpoints",
		},
		"closed endpoint": {
			current: map[*indicator.ContainerEndpoint]timestamp.MicroTS{
				endpoint1: past, // closed endpoint
			},
			expectCount: 1,
			description: "Should send closed endpoints",
		},
		"no endpoints": {
			current:     map[*indicator.ContainerEndpoint]timestamp.MicroTS{},
			expectCount: 0,
			description: "Should handle empty input",
		},
	}

	implementations := map[string]UpdateComputer{
		"Legacy":      NewLegacy(),
		"Categorized": NewCategorized(),
	}

	for implName, impl := range implementations {
		t.Run(implName, func(t *testing.T) {
			for name, tc := range testCases {
				t.Run(name, func(t *testing.T) {
					updates := impl.ComputeUpdatedEndpoints(tc.current)
					assert.Len(t, updates, tc.expectCount, tc.description)

					// Verify protobuf conversion
					for _, update := range updates {
						assert.NotNil(t, update.Props)
						assert.Equal(t, uint32(8080), update.Props.Port)
						assert.Equal(t, storage.L4Protocol_L4_PROTOCOL_TCP, update.Props.L4Protocol)
					}
				})
			}
		})
	}
}

// TestComputeUpdatedProcesses tests process update computation for both implementations
func TestComputeUpdatedProcesses(t *testing.T) {
	process1 := &indicator.ProcessListening{
		Process: indicator.ProcessInfo{
			ProcessName: "nginx",
			ProcessArgs: "-g daemon off;",
			ProcessExec: "/usr/sbin/nginx",
		},
		PodID:         "pod-1",
		ContainerName: "nginx-container",
		DeploymentID:  "nginx-deployment",
		PodUID:        "uid-123",
		Namespace:     "default",
		Port:          80,
		Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000

	testCases := map[string]struct {
		current     map[*indicator.ProcessListening]timestamp.MicroTS
		description string
	}{
		"new process": {
			current: map[*indicator.ProcessListening]timestamp.MicroTS{
				process1: now,
			},
			description: "Should handle new processes",
		},
		"closed process": {
			current: map[*indicator.ProcessListening]timestamp.MicroTS{
				process1: past, // closed process
			},
			description: "Should handle closed processes",
		},
		"no processes": {
			current:     map[*indicator.ProcessListening]timestamp.MicroTS{},
			description: "Should handle empty input",
		},
	}

	implementations := map[string]UpdateComputer{
		"Legacy":      NewLegacy(),
		"Categorized": NewCategorized(),
	}

	for implName, impl := range implementations {
		t.Run(implName, func(t *testing.T) {
			for name, tc := range testCases {
				t.Run(name, func(t *testing.T) {
					updates := impl.ComputeUpdatedProcesses(tc.current)

					// The actual behavior depends on the ProcessesListeningOnPort feature flag
					// We just ensure no panics and verify structure when updates exist
					for _, update := range updates {
						assert.NotNil(t, update.Process)
						assert.Equal(t, uint32(80), update.Port)
						assert.Equal(t, storage.L4Protocol_L4_PROTOCOL_TCP, update.Protocol)
						assert.Equal(t, "nginx", update.Process.ProcessName)
					}
				})
			}
		})
	}
}

// TestStateManagement tests ResetState and GetStateMetrics
func TestStateManagement(t *testing.T) {
	implementations := map[string]UpdateComputer{
		"Legacy":      NewLegacy(),
		"Categorized": NewCategorized(),
	}

	for implName, impl := range implementations {
		t.Run(implName, func(t *testing.T) {
			// Setup some state first
			entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
			conn1 := &indicator.NetworkConn{
				SrcEntity: entity1,
				DstEntity: entity1,
				DstPort:   80,
				Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
			}

			current := map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.Now(),
			}

			// Compute some updates to build state
			updates := impl.ComputeUpdatedConns(current)
			assert.NotEmpty(t, updates, "Should have initial updates")

			// Update state
			impl.UpdateState(current, nil, nil)

			// Test GetStateMetrics
			connsSize, endpointsSize, processesSize := impl.GetStateMetrics()

			// Test ResetState
			impl.ResetState()

			// Check state is cleared
			connsSize, endpointsSize, processesSize = impl.GetStateMetrics()
			assert.Equal(t, 0, connsSize, "Connections should be reset")
			assert.Equal(t, 0, endpointsSize, "Endpoints should be reset")
			assert.Equal(t, 0, processesSize, "Processes should be reset")
		})
	}
}

// TestCategorizedClosedConnectionTracking tests the new closed connection tracking functionality
func TestCategorizedClosedConnectionTracking(t *testing.T) {
	categorized := NewCategorized()

	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	conn1 := &indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity1,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000
	connKey := conn1.Key()

	t.Run("lookup_timestamp_for_open_connection", func(t *testing.T) {
		// Open connections should always return InfiniteFuture
		found, prevTS := categorized.lookupPrevTimestamp(connKey, timestamp.InfiniteFuture)
		assert.False(t, found, "Open connections should not be found in closed connection tracking")
		assert.Equal(t, timestamp.InfiniteFuture, prevTS, "Open connections should return InfiniteFuture")
	})

	t.Run("store_and_lookup_closed_connection", func(t *testing.T) {
		// Store a closed connection
		categorized.storeClosedConnectionTimestamp(connKey, past)

		// Look it up
		found, prevTS := categorized.lookupPrevTimestamp(connKey, now)
		assert.True(t, found, "Closed connection should be found in tracking")
		assert.Equal(t, past, prevTS, "Should return stored timestamp for closed connection")
	})

	t.Run("lookup_unknown_closed_connection", func(t *testing.T) {
		unknownKey := "unknown-connection"
		found, prevTS := categorized.lookupPrevTimestamp(unknownKey, now)
		assert.False(t, found, "Unknown connections should not be found in tracking")
		assert.Equal(t, timestamp.InfiniteFuture, prevTS, "Unknown connections should default to InfiniteFuture")
	})

	t.Run("cleanup_expired_connections", func(t *testing.T) {
		// Test cleanup by forcing a cleanup cycle
		categorized.lastCleanup = time.Now().Add(-2 * time.Minute)
		categorized.cleanupExpiredClosedConnections()
		// Should not panic and should update lastCleanup
	})
}

// TestCategorized_CategorizationEdgeCases tests edge cases in categorization logic
func TestCategorized_CategorizationEdgeCases(t *testing.T) {
	categorized := NewCategorized()

	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	conn1 := &indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity1,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000
	future := now + 1000

	testCases := map[string]struct {
		currTS   timestamp.MicroTS
		prevTS   timestamp.MicroTS
		expected UpdateCategory
	}{
		"connection_closed_transition": {
			currTS:   past,
			prevTS:   timestamp.InfiniteFuture,
			expected: RequiredUpdate,
		},
		"older_timestamp": {
			currTS:   past,
			prevTS:   now,
			expected: SkipUpdate,
		},
		"same_timestamp": {
			currTS:   now,
			prevTS:   now,
			expected: SkipUpdate,
		},
		"newer_timestamp_for_closed_connection": {
			currTS:   future,
			prevTS:   now,
			expected: SkipUpdate, // Subsequent updates for open connections are skipped
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			found, prevTS := categorized.lookupPrevTimestamp(conn1.Key(), tc.currTS)
			category := categorized.categorizeConnectionUpdate(conn1, tc.currTS, prevTS, found)
			assert.Equal(t, tc.expected, category, "Category should match expected")
		})
	}
}

// TestCategorized_ConditionalUpdateLogic tests the complete update logic including firstTimeSeen
func TestCategorized_ConditionalUpdateLogic(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	conn1 := &indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity1,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	closedRecently := now - 100
	closedLongAgo := now - 2000
	_ = closedLongAgo
	_ = closedRecently

	conditionalUpdateCases := map[string]struct {
		initialState  map[*indicator.NetworkConn]timestamp.MicroTS
		currentState  map[*indicator.NetworkConn]timestamp.MicroTS
		expectedCount int
	}{
		"new closed connection should always be sent as required update": {
			initialState: nil, // No initial state
			currentState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			expectedCount: 1,
		},
		"duplicate updates for closed connection with same timestamp should be skipped": {
			initialState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			currentState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			expectedCount: 0,
		},
		"recent updates for closed connection with younger close timestamps should be sent": {
			initialState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo, // Closed connection, but long ago
			},
			currentState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection, but younger than the previous close
			},
			expectedCount: 1,
		},
		"recent updates for closed connection with older close timestamps should be ignored": {
			initialState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection, but younger than the previous close
			},
			currentState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo, // Closed connection, but older than the previous close
			},
			expectedCount: 0,
		},
		"new open connections should be sent as required update": {
			initialState: nil, // No initial state
			currentState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // Open connection
			},
			expectedCount: 1,
		},
	}

	for name, tc := range conditionalUpdateCases {
		for implName, impl := range map[string]UpdateComputer{
			"Legacy":      NewLegacy(),
			"Categorized": NewCategorized(),
		} {
			t.Run(implName+"_"+name, func(t *testing.T) {
				computer := impl

				// Setup initial state if provided
				if tc.initialState != nil {
					if implName == "Legacy" {
						computer.UpdateState(tc.initialState, nil, nil)
					} else {
						computer.ComputeUpdatedConns(tc.initialState)
					}
				}

				updates := computer.ComputeUpdatedConns(tc.currentState)
				assert.Len(t, updates, tc.expectedCount, implName)
			})
		}
	}
}

// TestCategorizedRemovalBehavior tests the specific removal behaviors for open vs closed connections.
// This test ensures that the categorized update computer does not send updates when we deliberately decide
// to not track a given connection anymore. This opens up the possibility for Sensor to decide about deleting
// the connection from its state without notifying Central.
// This is not a problem, since we want to notify Central only in the following cases:
// - When we see a new connection
// - When we see a closed connection
// - When we see a connection that was previously open, but is now closed
// - When we see a connection that was previously closed, but is now closed with younger timestamp.
// In all other cases, we don't want to notify Central as there is no relevant change that affects any features.
func TestCategorizedRemovalBehavior(t *testing.T) {
	conn1 := &indicator.NetworkConn{
		SrcEntity: networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"},
		DstEntity: networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"},
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000

	removalCases := map[string]struct {
		previousState            map[*indicator.NetworkConn]timestamp.MicroTS
		currentState             map[*indicator.NetworkConn]timestamp.MicroTS
		expectLegacyRemoval      bool
		expectCategorizedRemoval bool
	}{
		"open connection disappearance: categorized should not send update": {
			previousState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // was open
			},
			currentState:             map[*indicator.NetworkConn]timestamp.MicroTS{}, // now gone
			expectLegacyRemoval:      true,                                           // Legacy tracks all removals
			expectCategorizedRemoval: false,                                          // Categorized doesn't track open connection removals
		},
		"closed connection disappearance: categorized should not send update": {
			previousState: map[*indicator.NetworkConn]timestamp.MicroTS{
				conn1: past, // was closed
			},
			currentState:             map[*indicator.NetworkConn]timestamp.MicroTS{}, // now gone
			expectLegacyRemoval:      true,                                           // Legacy tracks all removals
			expectCategorizedRemoval: false,                                          // Categorized should track closed connection removals
		},
	}

	for name, tc := range removalCases {
		t.Run(name, func(t *testing.T) {
			// Test Legacy
			t.Run("Legacy", func(t *testing.T) {
				legacy := NewLegacy()
				legacy.UpdateState(tc.previousState, nil, nil)
				results := legacy.ComputeUpdatedConns(tc.currentState)
				if tc.expectLegacyRemoval {
					assert.Len(t, results, 1, "Legacy: %s", name)
				} else {
					assert.Len(t, results, 0, "Legacy: %s", name)
				}
			})

			// Test Categorized
			t.Run("Categorized", func(t *testing.T) {
				categorized := NewCategorized()

				// First, establish the previous state by processing it
				categorized.ComputeUpdatedConns(tc.previousState)

				// Then check what happens when connections are removed
				results := categorized.ComputeUpdatedConns(tc.currentState)
				if tc.expectCategorizedRemoval {
					assert.Len(t, results, 1, "Categorized: %s", name)
				} else {
					assert.Len(t, results, 0, "Categorized: %s", name)
				}
			})
		})
	}
}
