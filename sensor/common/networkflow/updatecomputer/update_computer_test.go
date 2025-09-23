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

const (
	implLegacy          = "Legacy"
	implTransitionBased = "TransitionBased"
)

var closedConnRememberDuration = 5 * time.Minute

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
	open := timestamp.InfiniteFuture

	// We want to notify Central only in the following cases:
	// - When we see a new connection
	// - When we see a closed connection
	// - When we see a connection that was previously open, but is now closed
	// - When we see a connection that was previously closed, but is now closed with a newer timestamp.
	// In all other cases, we don't need to notify Central since there's no relevant change that affects any features -
	// including situations when a previously opened connection disappears.
	// Any unnecessary notification would be treated by Central as redundant and
	// consumes additional resources (network bandwidth between Sensor and Central, plus Central's CPU and memory).
	tests := map[string]struct {
		initialState     map[indicator.NetworkConn]timestamp.MicroTS
		currentState     map[indicator.NetworkConn]timestamp.MicroTS
		expectNumUpdates map[string]int
	}{
		// Test cases for scenarios most frequently observed in production
		// (i.e., a connection is being closed, or continues to be open).
		"should send when connection closes": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInThePast,
			},
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"closing connection in the future should be treated as any other update about connection closing": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInTheFuture,
			},
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"should not send duplicate open connections": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			expectNumUpdates: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		// Test cases for disappearance: when a connection that was open in the last state is gone without seeing a close message from Collector.
		// Correctly handling disappearance is crucial for allowing
		// Sensor to delete a connection from its state without notifying Central.
		"disappearance of open connection: legacy should send an update": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{},
			// Legacy tracks deletions and still produces a message (undesired behavior).
			// TransitionBased does not trigger an update (desired behavior).
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 0},
		},
		"disappearance of closed connection: legacy should send an update": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedInThePast,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{},
			// Legacy tracks deletions and still produces a message (undesired behavior).
			// TransitionBased does not trigger an update (desired behavior).
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 0},
		},
		"handling nils": {
			initialState:     nil,
			currentState:     nil,
			expectNumUpdates: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		// Test cases for empty initial state - behavior when a connection is seen for the first time.
		"new closed connection should always be sent as required update": {
			initialState: nil,
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently,
			},
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"new open connections should be sent as required update": {
			initialState: nil,
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: open,
			},
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		// Test cases for handling multiple messages about closing the same connection
		"duplicate updates for closed connection with same timestamp should be skipped": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently,
			},
			expectNumUpdates: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		"recent updates for closed connection with newer close timestamps should be sent": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently,
			},
			expectNumUpdates: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"recent updates for closed connection with older close timestamps should be ignored": {
			initialState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedRecently,
			},
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: closedLongAgo,
			},
			expectNumUpdates: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Run(implLegacy, func(t *testing.T) {
				computer := NewLegacy()
				if tc.initialState != nil {
					computer.OnSuccessfulSendConnections(tc.initialState)
				}
				// Call to OnSuccessfulSendConnections with nils should not change anything in the state
				computer.OnSuccessfulSendConnections(nil)
				updates := computer.ComputeUpdatedConns(tc.currentState)
				assert.Len(t, updates, tc.expectNumUpdates[implLegacy])
			})
			t.Run(implTransitionBased, func(t *testing.T) {
				computer := NewTransitionBased()
				if tc.initialState != nil {
					// Trigger a computation + successful send to bring the update computer to the initial state.
					computer.ComputeUpdatedConns(tc.initialState)
					computer.OnSuccessfulSendConnections(tc.initialState)
				}
				// Call to OnSuccessfulSendConnections with nils should not change anything in the state
				computer.OnSuccessfulSendConnections(nil)
				updates := computer.ComputeUpdatedConns(tc.currentState)
				assert.Len(t, updates, tc.expectNumUpdates[implTransitionBased])
			})
		})
	}
}

// Test_lookupPrevTimestamp tests the new closed connection tracking functionality
func Test_lookupPrevTimestamp(t *testing.T) {
	transitionBased := NewTransitionBased()

	nowTS := timestamp.Now()
	past := nowTS - 1000

	testCases := map[string]struct {
		connKey        string
		setupStore     func(name string)
		expectedFound  bool
		expectedPrevTS timestamp.MicroTS
	}{
		"Unknown connections should not be found and return 0": {
			connKey: "unknown-connection",
			setupStore: func(name string) {
				transitionBased.storeClosedConnectionTimestamp("foo-bar", past, closedConnRememberDuration)
			},
			expectedFound:  false,
			expectedPrevTS: 0,
		},
		"Open connections should not be found in closed connection tracking": {
			connKey:        "open-connection",
			setupStore:     func(_ string) {},
			expectedFound:  false,
			expectedPrevTS: 0,
		},
		"Stored closed connection should be found with correct timestamp": {
			connKey: "closed-connection-1",
			setupStore: func(name string) {
				transitionBased.storeClosedConnectionTimestamp(name, past, closedConnRememberDuration)
			},
			expectedFound:  true,
			expectedPrevTS: past,
		},
		"Stored closed connection should be found regardless of current timestamp": {
			connKey: "closed-connection-2",
			setupStore: func(name string) {
				transitionBased.storeClosedConnectionTimestamp(name, past, closedConnRememberDuration)
			},
			expectedFound:  true,
			expectedPrevTS: past,
		},
		"Stored closed connection should be found even with same timestamp": {
			connKey: "closed-connection-3",
			setupStore: func(name string) {
				transitionBased.storeClosedConnectionTimestamp(name, past, closedConnRememberDuration)
			},
			expectedFound:  true,
			expectedPrevTS: past,
		},
		"Stored closed connection should still be found after cleanup": {
			connKey: "closed-connection-4",
			setupStore: func(name string) {
				transitionBased.storeClosedConnectionTimestamp(name, past, closedConnRememberDuration)
			},
			expectedFound:  true,
			expectedPrevTS: past,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Setup: store connection if needed
			if tc.setupStore != nil {
				tc.setupStore(tc.connKey)
			}

			// Test: lookup the connection
			found, prevTS := transitionBased.lookupPrevTimestamp(tc.connKey)

			// Assertions
			assert.Equal(t, tc.expectedFound, found)
			assert.Equal(t, tc.expectedPrevTS, prevTS)
		})
	}

	// Additional test for cleanup functionality
	t.Run("should_not_panic_during_cleanup", func(t *testing.T) {
		now := time.Now()
		// Force cleanup by setting lastCleanup to a time in the past
		transitionBased.lastCleanup = now.Add(-2 * time.Minute)
		transitionBased.PeriodicCleanup(now, time.Minute)
		// Should not panic and should update lastCleanup
	})
}

// Test_lookupPrevTimestamp tests the new closed connection tracking functionality
func Test_closedConnTimestamps(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}
	conn1 := indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	nowGo := time.Now()
	nowTS := timestamp.FromGoTime(nowGo)

	testCases := map[string]struct {
		connKey        string
		currentState   map[indicator.NetworkConn]timestamp.MicroTS
		nowTS          time.Time
		rememberPeriod time.Duration
		expectedLength int
	}{
		"Closed connection should be remembered for at least 1000s": {
			connKey: "conn1",
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: nowTS.Add(-1000 * time.Second),
			},
			nowTS:          nowGo,
			rememberPeriod: 2000 * time.Second,
			expectedLength: 1,
		},
		"Closed connection should be forgotten after rememberPeriod": {
			connKey: "conn1",
			currentState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn1: nowTS.Add(-1000 * time.Second),
			},
			nowTS:          nowGo,
			rememberPeriod: 500 * time.Second,
			expectedLength: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			transitionBased := NewTransitionBased()
			transitionBased.closedConnRememberDuration = tc.rememberPeriod

			_ = transitionBased.ComputeUpdatedConns(tc.currentState)
			transitionBased.PeriodicCleanup(tc.nowTS, 0)
			assert.Equal(t, tc.expectedLength, len(transitionBased.closedConnTimestamps))
		})
	}
}

func TestComputeUpdatedEndpointsAndProcesses(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}

	ep1 := indicator.ContainerEndpoint{
		Entity:   entity1,
		Port:     8080,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	p1 := indicator.ProcessListening{
		Process: indicator.ProcessInfo{
			ProcessName: "proc-name",
			ProcessArgs: "proc-args",
			ProcessExec: "proc-exec",
		},
		PodID:         "",
		ContainerName: "",
		DeploymentID:  "",
		PodUID:        "",
		Namespace:     "",
		Protocol:      0,
		Port:          0,
	}
	p2 := indicator.ProcessListening{
		Process: indicator.ProcessInfo{
			ProcessName: "proc2-name",
			ProcessArgs: "proc2-args",
			ProcessExec: "proc2-exec",
		},
		PodID:         "",
		ContainerName: "",
		DeploymentID:  "",
		PodUID:        "",
		Namespace:     "",
		Protocol:      0,
		Port:          0,
	}

	closedNow := timestamp.Now()
	open := timestamp.InfiniteFuture
	closedInThePast := closedNow - 1000

	testCases := map[string]struct {
		initialMapping       map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose
		currentMapping       map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose
		expectNumUpdatesEp   map[string]int
		expectNumUpdatesProc map[string]int
	}{
		"Should send new closed endpoints": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedNow,
				},
			},
			expectNumUpdatesEp:   map[string]int{implLegacy: 1, implTransitionBased: 1},
			expectNumUpdatesProc: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"Should send update when open endpoints are closed": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedInThePast,
				},
			},
			expectNumUpdatesEp:   map[string]int{implLegacy: 1, implTransitionBased: 1},
			expectNumUpdatesProc: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"Should not send an update when open endpoints remain open": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			expectNumUpdatesEp:   map[string]int{implLegacy: 0, implTransitionBased: 0},
			expectNumUpdatesProc: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		"Should not send update when closed TS is updated to a past value": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedNow,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedInThePast,
				},
			},
			// We do not track close-timestamps for endpoints as we do for connections.
			// This results in always sending updates on closed->closed transitions.
			// This is intentional, as we estimate lower overhead in sending duplicates compared to
			// tracking all closed endpoints in memory for a limited time (as done for connections).
			expectNumUpdatesEp:   map[string]int{implLegacy: 0, implTransitionBased: 1},
			expectNumUpdatesProc: map[string]int{implLegacy: 0, implTransitionBased: 1},
		},
		"Should send update when closed TS is updated to a younger value": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedInThePast,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         closedNow,
				},
			},
			expectNumUpdatesEp:   map[string]int{implLegacy: 1, implTransitionBased: 1},
			expectNumUpdatesProc: map[string]int{implLegacy: 1, implTransitionBased: 1},
		},
		"Should produce no updates on empty input": {
			initialMapping:       map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{},
			currentMapping:       map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{},
			expectNumUpdatesEp:   map[string]int{implLegacy: 0, implTransitionBased: 0},
			expectNumUpdatesProc: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		"Should send an update on deletion for legacy but not for TransitionBased": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			currentMapping:       map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{},
			expectNumUpdatesEp:   map[string]int{implLegacy: 1, implTransitionBased: 0},
			expectNumUpdatesProc: map[string]int{implLegacy: 1, implTransitionBased: 0},
		},
		"handling nils": {
			initialMapping:       nil,
			currentMapping:       nil,
			expectNumUpdatesEp:   map[string]int{implLegacy: 0, implTransitionBased: 0},
			expectNumUpdatesProc: map[string]int{implLegacy: 0, implTransitionBased: 0},
		},
		// Process-specific cases
		"Should replace process on two open-endpoint messages": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p2,
					LastSeen:         open,
				},
			},
			expectNumUpdatesEp: map[string]int{implLegacy: 0, implTransitionBased: 0},
			// Legacy sends 2 updates, because first is for p1 to disappear, and second is for p2 to appear.
			expectNumUpdatesProc: map[string]int{implLegacy: 2, implTransitionBased: 1},
		},
		"A replacement triggered with a close message should close the correct process": {
			initialMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p1,
					LastSeen:         open,
				},
			},
			currentMapping: map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{
				ep1: {
					ProcessListening: &p2,
					LastSeen:         closedNow,
				},
			},
			expectNumUpdatesEp: map[string]int{implLegacy: 1, implTransitionBased: 1}, // for closing ep1
			// Legacy sends 2 updates, because first is for p1 to disappear, and second is for p2 to appear (as closed).
			expectNumUpdatesProc: map[string]int{implLegacy: 2, implTransitionBased: 1},
		},
	}

	executeAssertions := func(t *testing.T, l UpdateComputer,
		expectedNumUpdatesEp, expectedNumUpdatesProc int,
		initialMapping, currentMapping map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose) {
		t.Helper()
		// Bring model to the initial state
		ie, ip := l.ComputeUpdatedEndpointsAndProcesses(initialMapping)
		_, _ = ie, ip
		l.OnSuccessfulSendEndpoints(initialMapping)
		l.OnSuccessfulSendProcesses(initialMapping)
		// Call to OnSuccessfulSend with nils should not change anything in the state
		l.OnSuccessfulSendEndpoints(nil)
		l.OnSuccessfulSendProcesses(nil)

		updatesEp, updatesProc := l.ComputeUpdatedEndpointsAndProcesses(currentMapping)
		assert.Len(t, updatesEp, expectedNumUpdatesEp, "num endpoint updates incorrect")
		assert.Len(t, updatesProc, expectedNumUpdatesProc, "num proc updates incorrect")

		// Verify protobuf conversion
		for _, update := range updatesEp {
			assert.NotNil(t, update.Props)
			assert.Equal(t, uint32(8080), update.Props.Port)
			assert.Equal(t, storage.L4Protocol_L4_PROTOCOL_TCP, update.Props.L4Protocol)
		}
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Run(implLegacy, func(t *testing.T) {
				executeAssertions(t, NewLegacy(),
					tc.expectNumUpdatesEp[implLegacy], tc.expectNumUpdatesProc[implLegacy],
					tc.initialMapping, tc.currentMapping)
			})
			t.Run(implTransitionBased, func(t *testing.T) {
				executeAssertions(t, NewTransitionBased(),
					tc.expectNumUpdatesEp[implTransitionBased], tc.expectNumUpdatesProc[implTransitionBased],
					tc.initialMapping, tc.currentMapping)
			})
		})
	}
}
