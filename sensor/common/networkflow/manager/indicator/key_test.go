package indicator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stretchr/testify/assert"
)

func TestNetworkConn_KeyGeneration(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

	tests := map[string]NetworkConn{
		"Same indicators should produce identical keys: TCP connection": {
			SrcEntity: entity1,
			DstEntity: entity2,
			DstPort:   80,
			Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		"Same indicators should produce identical keys: UDP connection": {
			SrcEntity: entity2,
			DstEntity: entity1,
			DstPort:   53,
			Protocol:  storage.L4Protocol_L4_PROTOCOL_UDP,
		},
		"Same indicators should produce identical keys: High port number": {
			SrcEntity: entity1,
			DstEntity: entity2,
			DstPort:   65535,
			Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	for name, conn := range tests {
		t.Run(name, func(t *testing.T) {
			conn1, conn2 := conn, conn
			key1Hash := conn1.Key()
			key2Hash := conn2.Key()
			assert.Equal(t, key1Hash, key2Hash)
			assert.Len(t, key1Hash, 16)

			// Additional validation using the hash method directly
			assert.Equal(t, conn1.keyHash(), conn2.keyHash())
		})
	}
}

func TestContainerEndpoint_KeyGeneration(t *testing.T) {
	entity := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}

	tests := map[string]ContainerEndpoint{
		"Same indicators should produce identical keys: HTTP endpoint": {
			Entity:   entity,
			Port:     8080,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		"Same indicators should produce identical keys: DNS endpoint": {
			Entity:   entity,
			Port:     53,
			Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
		},
		"Same indicators should produce identical keys: HTTPS endpoint": {
			Entity:   entity,
			Port:     443,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	for name, endpoint := range tests {
		t.Run(name, func(t *testing.T) {
			ep1, ep2 := endpoint, endpoint
			key1Hash := ep1.BinaryKey()
			key2Hash := ep2.BinaryKey()
			assert.Equal(t, key1Hash, key2Hash)
			assert.NotZero(t, key1Hash)

			// Additional validation using the hash method directly
			assert.Equal(t, ep1.binaryKeyHash(), ep2.binaryKeyHash())
		})
	}
}

func TestProcessListening_KeyGeneration(t *testing.T) {
	tests := map[string]ProcessListening{
		"Same indicators should produce identical keys: Nginx process": {
			Process: ProcessInfo{
				ProcessName: "nginx",
				ProcessArgs: "-g daemon off;",
				ProcessExec: "/usr/sbin/nginx",
			},
			PodID:         "nginx-pod-123",
			ContainerName: "nginx-container",
			DeploymentID:  "nginx-deployment",
			PodUID:        "pod-uid-456",
			Namespace:     "default",
			Port:          80,
			Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		"Same indicators should produce identical keys: Redis process": {
			Process: ProcessInfo{
				ProcessName: "redis-server",
				ProcessArgs: "/etc/redis/redis.conf",
				ProcessExec: "/usr/bin/redis-server",
			},
			PodID:         "redis-pod-789",
			ContainerName: "redis-container",
			DeploymentID:  "redis-deployment",
			PodUID:        "pod-uid-101112",
			Namespace:     "cache",
			Port:          6379,
			Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	for name, process := range tests {
		t.Run(name, func(t *testing.T) {
			proc1, proc2 := process, process
			key1Hash := proc1.BinaryKey()
			key2Hash := proc2.BinaryKey()
			assert.Equal(t, key1Hash, key2Hash)
			assert.NotZero(t, key1Hash)

			// Additional validation using the hash method directly
			assert.Equal(t, proc1.binaryKeyHash(), proc2.binaryKeyHash())
		})
	}
}

func TestKey_UniquenessForDifferentObjects(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

	t.Run("connections", func(t *testing.T) {
		// Test that different NetworkConn objects produce different keys
		conn1 := NetworkConn{
			SrcEntity: entity1,
			DstEntity: entity2,
			DstPort:   80,
			Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
		}
		conn2 := NetworkConn{
			SrcEntity: entity1,
			DstEntity: entity2,
			DstPort:   443, // Different port
			Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
		}
		assert.NotEqual(t, conn1.Key(), conn2.Key(),
			"Different NetworkConn objects should have different keys")
	})

	t.Run("endpoints", func(t *testing.T) {
		// Test that different ContainerEndpoint objects produce different keys
		ep1 := ContainerEndpoint{
			Entity:   entity1,
			Port:     8080,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}
		ep2 := ContainerEndpoint{
			Entity:   entity1,
			Port:     8080,
			Protocol: storage.L4Protocol_L4_PROTOCOL_UDP, // Different protocol
		}

		assert.NotEqual(t, ep1.BinaryKey(), ep2.BinaryKey(),
			"Different ContainerEndpoint objects should have different binary hash keys")
	})
}
func TestKey_UniquenessForProcessListening(t *testing.T) {
	cases := map[string]struct {
		p1                ProcessListening
		p2                ProcessListening
		expectEqualHashes bool
	}{
		"Different processes should have different hashes": {
			p1: ProcessListening{
				Process: ProcessInfo{
					ProcessName: "hack",
					ProcessArgs: "--all",
					ProcessExec: "/usr/sbin/hack",
				},
				PodID:         "boom1",
				ContainerName: "hack-container",
				DeploymentID:  "abc",
				PodUID:        "efg",
				Namespace:     "default",
				Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
				Port:          80,
			},
			p2: ProcessListening{
				Process: ProcessInfo{
					ProcessName: "nginx",
					ProcessArgs: "--port 8080",
					ProcessExec: "/usr/bin/nginx",
				},
				PodID:         "nginx-pod-123",
				ContainerName: "nginx-container",
				DeploymentID:  "abc",
				PodUID:        "efg",
				Namespace:     "default",
				Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
				Port:          8080,
			},
			expectEqualHashes: false,
		},
		"Changes in non-key fields should produce same hashes": {
			p1: ProcessListening{
				Process: ProcessInfo{ // key field
					ProcessName: "hack",
					ProcessArgs: "--all",
					ProcessExec: "/usr/sbin/hack",
				},
				PodID:         "boom1",          // key field
				ContainerName: "hack-container", // key field
				DeploymentID:  "abc",
				PodUID:        "efg",
				Namespace:     "default",
				Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP, // key field
				Port:          80,                                 // key field
			},
			p2: ProcessListening{
				Process: ProcessInfo{ // key field
					ProcessName: "hack",
					ProcessArgs: "--all",
					ProcessExec: "/usr/sbin/hack",
				},
				PodID:         "boom1",          // key field
				ContainerName: "hack-container", // key field
				// The assumption is that if the 3 fields below are different,
				// then (in reality) the pod-ID must also be different.
				DeploymentID: "something-totally-different-than-in-p1",
				PodUID:       "something-totally-different-than-in-p1",
				Namespace:    "something-totally-different-than-in-p1",
				Protocol:     storage.L4Protocol_L4_PROTOCOL_TCP, // key field
				Port:         80,                                 // key field
			},
			expectEqualHashes: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var assertFunc func(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool
			assertFunc = assert.NotEqual
			textNugget := "different"
			if tc.expectEqualHashes {
				assertFunc = assert.Equal
				textNugget = "same"
			}
			assertFunc(t, tc.p1.BinaryKey(), tc.p2.BinaryKey(),
				"Different ProcessListening objects should have %s binary hash keys", textNugget)
		})
	}
}

func TestKeyUtilities_PortAndProtocolHandling(t *testing.T) {
	entity := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "test-deployment"}

	tests := map[string]struct {
		port     uint16
		protocol storage.L4Protocol
	}{
		"Min port":      {0, storage.L4Protocol_L4_PROTOCOL_TCP},
		"Max port":      {65535, storage.L4Protocol_L4_PROTOCOL_UDP},
		"Standard HTTP": {80, storage.L4Protocol_L4_PROTOCOL_TCP},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			endpoint := ContainerEndpoint{
				Entity:   entity,
				Port:     tc.port,
				Protocol: tc.protocol,
			}

			binaryHashKey := endpoint.BinaryKey()

			assert.NotZero(t, binaryHashKey)
		})
	}
}
