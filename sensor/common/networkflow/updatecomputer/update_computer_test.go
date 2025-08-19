package updatecomputer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
)

func TestComputeUpdatedConns(t *testing.T) {
	// Test data setup
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

	conn1 := indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	closedRecently := now - 100
	closedInThePast := now - 1000
	closedInTheFuture := now + 1000
	closedLongAgo := now - 2000

	// We want to notify Central only in the following cases:
	// - When we see a new connection
	// - When we see a closed connection
	// - When we see a connection that was previously open, but is now closed
	// - When we see a connection that was previously closed, but is now closed with younger timestamp.
	// In all other cases, we don't need to notify Central as there is no relevant change that affects any features.
	// Any such notification would be treated by Central as redundant.
	tests := map[string]struct {
		initialState   map[indicator.NetworkConn]timestamp.MicroTS
		currentState   map[indicator.NetworkConn]timestamp.MicroTS
		expectedCount  int
		expectWarnings bool
	}{
		// Test-cases for: the most frequent scenarios, i.e., a connection being closed, or continues to be open.
		"should send when connection closes": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // was open
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInThePast, // closed connection
			},
			expectedCount: 1,
		},
		"closing connection in the future should be treated as any other update about connection closing": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInTheFuture,
			},
			expectedCount: 1,
		},
		"should not send duplicate open connections": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			expectedCount: 0,
		},
		// Test-cases for: disappearance, i.e., the current state is empty.
		// The disappearance tests ensure that the categorized update computer does not send updates when
		// we deliberately decide to not track a given connection anymore.
		// This opens up the possibility for Sensor to delete a connection from its state without notifying Central.
		"disappearance of open connection: categorized should not send update": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture,
			},
			currentState:   map[indicator.NetworkConn]timestamp.MicroTS{},
			expectedCount:  1,    // Legacy method would still produce a message
			expectWarnings: true, // Things should rather not disappear without message from collector - warning.
		},
		"disappearance of closed connection: categorized should not send update": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInThePast,
			},
			currentState:   map[indicator.NetworkConn]timestamp.MicroTS{},
			expectedCount:  1,    // Legacy method would still produce a message
			expectWarnings: true, // Things should rather not disappear without message from collector - warning.
		},
		"handling nils": {
			initialState:   nil,
			currentState:   nil,
			expectedCount:  0,
			expectWarnings: true,
		},
		// Test-cases for: Initial state is empty - behavior when a connection is seen for the first time.
		"new closed connection should always be sent as required update": {
			initialState: nil,
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			expectedCount: 1,
		},
		"new open connections should be sent as required update": {
			initialState: nil,
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: timestamp.InfiniteFuture, // Open connection
			},
			expectedCount: 1,
		},
		// Test-cases for: Handling similar messages for connection closing
		"duplicate updates for closed connection with same timestamp should be skipped": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection
			},
			expectedCount: 0,
		},
		"recent updates for closed connection with younger close timestamps should be sent": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo, // Closed connection, but long ago
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection, but younger than the previous close
			},
			expectedCount: 1,
		},
		"recent updates for closed connection with older close timestamps should be ignored": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently, // Closed connection, but younger than the previous close
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo, // Closed connection, but older than the previous close
			},
			expectedCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			computer := NewLegacy()
			if tc.initialState != nil {
				computer.UpdateState(tc.initialState, nil, nil)
			}
			// Legacy implementation never returns warnings (although it theoretically could)
			updates := computer.ComputeUpdatedConns(tc.currentState)
			assert.Len(t, updates, tc.expectedCount)
		})
	}
}

// TestComputeUpdatedEndpoints tests endpoint update computation for both implementations
func TestComputeUpdatedEndpoints(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}

	endpoint1 := indicator.ContainerEndpoint{
		Entity:   entity1,
		Port:     8080,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	now := timestamp.Now()
	past := now - 1000

	testCases := map[string]struct {
		current     map[indicator.ContainerEndpoint]timestamp.MicroTS
		expectCount int
	}{
		"Should send new closed endpoints": {
			current: map[indicator.ContainerEndpoint]timestamp.MicroTS{
				endpoint1: now,
			},
			expectCount: 1,
		},
		"Should send closed endpoints": {
			current: map[indicator.ContainerEndpoint]timestamp.MicroTS{
				endpoint1: past, // closed endpoint
			},
			expectCount: 1,
		},
		"Should produce no updates on empty input": {
			current:     map[indicator.ContainerEndpoint]timestamp.MicroTS{},
			expectCount: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			updates := NewLegacy().ComputeUpdatedEndpoints(tc.current)
			assert.Len(t, updates, tc.expectCount)

			// Verify protobuf conversion
			for _, update := range updates {
				assert.NotNil(t, update.Props)
				assert.Equal(t, uint32(8080), update.Props.Port)
				assert.Equal(t, storage.L4Protocol_L4_PROTOCOL_TCP, update.Props.L4Protocol)
			}
		})
	}
}

// TestComputeUpdatedProcesses tests process update computation for both implementations
func TestComputeUpdatedProcesses(t *testing.T) {
	process1 := indicator.ProcessListening{
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
		current     map[indicator.ProcessListening]timestamp.MicroTS
		description string
	}{
		"new process": {
			current: map[indicator.ProcessListening]timestamp.MicroTS{
				process1: now,
			},
			description: "Should handle new processes",
		},
		"closed process": {
			current: map[indicator.ProcessListening]timestamp.MicroTS{
				process1: past, // closed process
			},
			description: "Should handle closed processes",
		},
		"no processes": {
			current:     map[indicator.ProcessListening]timestamp.MicroTS{},
			description: "Should handle empty input",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			updates := NewLegacy().ComputeUpdatedProcesses(tc.current)

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
}
