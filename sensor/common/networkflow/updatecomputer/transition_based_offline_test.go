package updatecomputer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
)

// TestTransitionBasedComputeUpdatedConnsOffline tests the offline behavior for the TransitionBased update computer.
// This test doesn't apply to `Legacy` since its offline behavior relies on implementation details
// within NetFlowManager.
func TestTransitionBasedComputeUpdatedConnsOffline(t *testing.T) {
	// Test data setup
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}
	entity3 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-3"}
	entity4 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-4"}

	conn12 := indicator.NetworkConn{
		SrcEntity: entity1,
		DstEntity: entity2,
		DstPort:   8012,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn23 := indicator.NetworkConn{
		SrcEntity: entity2,
		DstEntity: entity3,
		DstPort:   8023,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn24 := indicator.NetworkConn{
		SrcEntity: entity2,
		DstEntity: entity4,
		DstPort:   8024,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
	conn34 := indicator.NetworkConn{
		SrcEntity: entity3,
		DstEntity: entity4,
		DstPort:   8034,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	emptyUpdate := map[indicator.NetworkConn]timestamp.MicroTS{}

	now := timestamp.Now()
	closedLongAgo := now - 2000
	closedInThePast := now - 1000
	closedRecently := now - 100
	closedInTheFuture := now + 1000
	open := timestamp.InfiniteFuture

	tests := map[string]struct {
		initialOnlineState map[indicator.NetworkConn]timestamp.MicroTS
		offlineUpdate1     map[indicator.NetworkConn]timestamp.MicroTS
		offlineUpdate2     map[indicator.NetworkConn]timestamp.MicroTS
		offlineUpdate3     map[indicator.NetworkConn]timestamp.MicroTS
		currentOnlineState map[indicator.NetworkConn]timestamp.MicroTS
		expectNumUpdates   int
	}{
		"new updates arriving while offline should be sent once transitioning to online": {
			initialOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn23: open,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn24: open,
			},
			offlineUpdate3: map[indicator.NetworkConn]timestamp.MicroTS{
				conn34: open,
			},
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedInThePast,
			},
			// 3 offline updates and 1 from `currentOnlineState`
			expectNumUpdates: 4,
		},
		"4 close messages with incrementing timestamps arriving when offline should produce 4 updates": {
			initialOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedLongAgo,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedInThePast,
			},
			offlineUpdate3: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedRecently,
			},
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedInTheFuture,
			},
			// If the update computer was online the entire time, it would also produce 4 updates.
			expectNumUpdates: 4,
		},
		"4 close messages with decrementing timestamps arriving when offline should produce 1 update": {
			initialOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedInTheFuture,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedRecently,
			},
			offlineUpdate3: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedInThePast,
			},
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedLongAgo,
			},
			// If the update computer was online the entire time, it would also produce 1 update.
			expectNumUpdates: 1,
		},
		"4 open messages for the same already known EE should produce 1 update": {
			initialOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate3: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			// We expect exactly the same behavior as being online the entire time.
			// The initial update is sent after `initialOnlineState`, but this test doesn't assert on the initial condition.
			// All offline updates and the `currentOnlineState` yield no new updates since the EE remains unchanged.
			expectNumUpdates: 0,
		},
		"4 open messages for the same unknown yet EE should produce 1 update": {
			initialOnlineState: emptyUpdate,
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate3: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			// We expect exactly the same behavior as being online the entire time.
			// Four identical open EEs should generate a single update that is sent when first going online.
			expectNumUpdates: 1,
		},
		"short-lived connection blip while offline should not be missed even if there are no updates after going online": {
			initialOnlineState: emptyUpdate,
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedRecently,
			},
			offlineUpdate3:     emptyUpdate,
			currentOnlineState: emptyUpdate,
			// One update for opening, one for closing.
			expectNumUpdates: 2,
		},
		"short-lived connection blip while offline should not be missed when other updates are present": {
			initialOnlineState: emptyUpdate,
			offlineUpdate1: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: open,
			},
			offlineUpdate2: map[indicator.NetworkConn]timestamp.MicroTS{
				conn12: closedRecently,
			},
			offlineUpdate3: emptyUpdate,
			currentOnlineState: map[indicator.NetworkConn]timestamp.MicroTS{
				conn34: closedRecently,
			},
			// One update for opening, one for closing.
			expectNumUpdates: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := NewTransitionBased()
			// Initial online update - for TransitionBased, we must trigger a single computation and call `OnSuccessfulSend`
			_ = l.ComputeUpdatedConns(tc.initialOnlineState)
			l.OnSuccessfulSendConnections(tc.initialOnlineState)

			// Going offline - calling ComputeUpdatedConns but not OnSuccessfulSend
			_ = l.ComputeUpdatedConns(tc.offlineUpdate1)
			_ = l.ComputeUpdatedConns(tc.offlineUpdate2)
			_ = l.ComputeUpdatedConns(tc.offlineUpdate3)

			// Going online again - calling ComputeUpdatedConns followed by OnSuccessfulSend
			finalUpdates := l.ComputeUpdatedConns(tc.currentOnlineState)
			l.OnSuccessfulSendConnections(tc.currentOnlineState)

			assert.Len(t, finalUpdates, tc.expectNumUpdates)

			// Empty update to ensure that any caches for offline mode are cleared
			u := l.ComputeUpdatedConns(emptyUpdate)
			l.OnSuccessfulSendConnections(emptyUpdate)
			assert.Len(t, u, 0)
		})
	}
}
