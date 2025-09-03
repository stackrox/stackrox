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

			key1String := conn1.Key(HashingAlgoString)
			key2String := conn2.Key(HashingAlgoString)
			assert.Equal(t, key1String, key2String)
			assert.NotEmpty(t, key1String)

			key1Hash := conn1.Key(HashingAlgoHash)
			key2Hash := conn2.Key(HashingAlgoHash)
			assert.Equal(t, key1Hash, key2Hash)
			assert.Len(t, key1Hash, 16)

			assert.NotEqual(t, key1String, key1Hash)
			assert.Equal(t, conn1.keyString(), conn2.keyString())
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

			key1String := ep1.Key(HashingAlgoString)
			key2String := ep2.Key(HashingAlgoString)
			assert.Equal(t, key1String, key2String)
			assert.NotEmpty(t, key1String)

			key1Hash := ep1.Key(HashingAlgoHash)
			key2Hash := ep2.Key(HashingAlgoHash)
			assert.Equal(t, key1Hash, key2Hash)
			assert.Len(t, key1Hash, 16)

			assert.NotEqual(t, key1String, key1Hash)
			assert.Equal(t, ep1.keyString(), ep2.keyString())
			assert.Equal(t, ep1.keyHash(), ep2.keyHash())
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

			key1String := proc1.Key(HashingAlgoString)
			key2String := proc2.Key(HashingAlgoString)
			assert.Equal(t, key1String, key2String)
			assert.NotEmpty(t, key1String)

			key1Hash := proc1.Key(HashingAlgoHash)
			key2Hash := proc2.Key(HashingAlgoHash)
			assert.Equal(t, key1Hash, key2Hash)
			assert.Len(t, key1Hash, 16)

			assert.NotEqual(t, key1String, key1Hash)
			assert.Equal(t, proc1.keyString(), proc2.keyString())
			assert.Equal(t, proc1.keyHash(), proc2.keyHash())
		})
	}
}

func TestKey_UniquenessForDifferentObjects(t *testing.T) {
	entity1 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-1"}
	entity2 := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "deployment-2"}

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

	assert.NotEqual(t, conn1.Key(HashingAlgoString), conn2.Key(HashingAlgoString),
		"Different NetworkConn objects should have different string keys")
	assert.NotEqual(t, conn1.Key(HashingAlgoHash), conn2.Key(HashingAlgoHash),
		"Different NetworkConn objects should have different hash keys")

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

	assert.NotEqual(t, ep1.Key(HashingAlgoString), ep2.Key(HashingAlgoString),
		"Different ContainerEndpoint objects should have different string keys")
	assert.NotEqual(t, ep1.Key(HashingAlgoHash), ep2.Key(HashingAlgoHash),
		"Different ContainerEndpoint objects should have different hash keys")
}

func TestHashingAlgo_DefaultBehavior(t *testing.T) {
	entity := networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: "test-deployment"}
	conn := NetworkConn{
		SrcEntity: entity,
		DstEntity: entity,
		DstPort:   80,
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}

	// Test that unknown hashing algorithm defaults to hash
	unknownAlgo := HashingAlgo(99)
	keyWithUnknown := conn.Key(unknownAlgo)
	keyWithHash := conn.Key(HashingAlgoHash)

	assert.Equal(t, keyWithUnknown, keyWithHash,
		"Unknown hashing algorithm should default to hash algorithm")
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

			stringKey := endpoint.Key(HashingAlgoString)
			hashKey := endpoint.Key(HashingAlgoHash)

			assert.NotEmpty(t, stringKey)
			assert.NotEmpty(t, hashKey)
			assert.Len(t, hashKey, 16)
		})
	}
}
