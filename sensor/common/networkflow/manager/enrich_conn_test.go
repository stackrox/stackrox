package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestHandleContainerNotFound(t *testing.T) {
	contIDGraceSeconds := int64(env.ContainerIDResolutionGracePeriod.DurationSetting().Seconds())
	exampleConn := &connection{
		local: net.NetworkPeerID{
			Address:   net.ParseIP("192.168.111.11"),
			Port:      90,
			IPNetwork: net.IPNetworkFromCIDR("192.168.111.0/24"),
		},
		remote: net.NumericEndpoint{
			IPAndPort: net.NetworkPeerID{
				Address:   net.ParseIP("192.168.111.99"),
				Port:      90,
				IPNetwork: net.IPNetworkFromCIDR("192.168.111.0/24"),
			},
			L4Proto: 0, // TCP
		},
		containerID: "dummy",
		incoming:    false,
	}
	exampleConnIndicator := &networkConnIndicator{
		srcEntity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   "abc",
		},
		dstEntity: networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   "efg",
		},
		dstPort:  90,
		protocol: 0, // TCP
	}

	testCases := map[string]struct {
		conn      *connection
		firstSeen timestamp.MicroTS
		now       timestamp.MicroTS
		isActive  bool
		// expectations
		wantClosing        bool
		wantRot            bool
		wantNumFailReasons int
		wantErrContain     []string
	}{
		"active connection within the grace period should not be marked as closed": {
			conn:      exampleConn,
			firstSeen: 80,
			now:       81,
			isActive:  true,

			wantClosing:        false,
			wantRot:            false,
			wantNumFailReasons: 2,
			wantErrContain: []string{
				"ContainerID dummy unknown",
				"time for container resolution", "not elapsed yet",
			},
		},
		"active connection outside the grace period should be marked for closing": {
			conn:      exampleConn,
			firstSeen: timestamp.FromGoTime(time.Unix(80, 0)),
			now:       timestamp.FromGoTime(time.Unix(80+contIDGraceSeconds+1, 0)),
			isActive:  true,

			wantClosing:        true,
			wantRot:            false,
			wantNumFailReasons: 0, // connection is active, so we will close it - this yields no failure reason, because it is not considered a failure
			wantErrContain:     []string{},
		},
		"inactive connection outside the grace period should be marked for closing": {
			conn:      exampleConn,
			firstSeen: timestamp.FromGoTime(time.Unix(80, 0)),
			now:       timestamp.FromGoTime(time.Unix(80+contIDGraceSeconds+1, 0)),
			isActive:  false,

			wantClosing:        false,
			wantRot:            true,
			wantNumFailReasons: 1,
			wantErrContain: []string{
				"ContainerID dummy unknown",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			activeConnections := make(map[connection]*networkConnIndicator)
			if tc.isActive {
				activeConnections[*tc.conn] = exampleConnIndicator
			}
			activeToClose, rotten, reason := handleContainerNotFound(
				tc.conn, tc.firstSeen, activeConnections, tc.now)
			assert.Equal(t, tc.wantClosing, activeToClose != nil, "expectation for closing the connection is not met")
			assert.Equal(t, tc.wantRot, rotten, "expectation for marking the connection as rotten not met")

			for _, msg := range tc.wantErrContain {
				assert.ErrorContains(t, reason, msg)
			}
			switch len(tc.wantErrContain) {
			case 0:
				assert.Nil(t, reason)
			case 1:
				assert.NotNil(t, reason)
			default:
				assert.NotNil(t, reason)
				assert.ErrorContains(t, reason, fmt.Sprintf("%d errors occurred:", tc.wantNumFailReasons))
			}
		})
	}

}
