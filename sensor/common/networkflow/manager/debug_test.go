package manager

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/pkg/net"
)

func TestDebug(t *testing.T) {

	hc := make(map[string]*hostConnections)
	hc["1"] = &hostConnections{
		hostname: "xxx",
		connections: map[connection]*connStatus{
			{
				local: net.NetworkPeerID{
					Address:   net.ParseIP("172.0.0.1"),
					Port:      0,
					IPNetwork: net.IPNetworkFromCIDR("192.168.0.0/24"),
				},
				remote: net.NumericEndpoint{
					IPAndPort: net.NetworkPeerID{
						Address:   net.ParseIP("172.0.0.1"),
						Port:      0,
						IPNetwork: net.IPNetworkFromCIDR("192.168.0.0/24"),
					},
					L4Proto: 0,
				},
				containerID: "container1",
				incoming:    false,
			}: {
				firstSeen:   0,
				lastSeen:    0,
				used:        false,
				usedProcess: false,
				rotten:      false,
			},
		},
		endpoints: map[containerEndpoint]*connStatus{
			{
				endpoint: net.NumericEndpoint{
					IPAndPort: net.NetworkPeerID{
						Address:   net.ParseIP("172.0.0.1"),
						Port:      0,
						IPNetwork: net.IPNetworkFromCIDR("192.168.0.0/24"),
					},
					L4Proto: 0,
				},
				containerID: "container2",
				processKey: processInfo{
					processName: "process1",
					processArgs: "arg0",
					processExec: "exec5",
				},
			}: {
				firstSeen:   0,
				lastSeen:    0,
				used:        false,
				usedProcess: false,
				rotten:      false,
			},
		},
		lastKnownTimestamp:    0,
		connectionsSequenceID: 0,
		currentSequenceID:     0,
		pendingDeletion:       nil,
	}
	data, err := json.Marshal(hc)
	if err != nil {
		t.Errorf("marshalling error: %v", err)
	}
	t.Logf("%s", data)
}
