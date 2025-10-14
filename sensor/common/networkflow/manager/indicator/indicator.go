package indicator

import (
	"fmt"
	"hash"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

// BinaryHash represents a 64-bit hash for memory-efficient key storage.
// Using uint64 directly avoids conversion overhead and provides faster map operations
// compared to [8]byte (single-instruction comparison vs byte-by-byte).
// Switching to a 128-bit hash would require using [16]byte.
type BinaryHash uint64

// ProcessInfo represents process information used in indicators
type ProcessInfo struct {
	ProcessName string
	ProcessArgs string
	ProcessExec string
}

func (p *ProcessInfo) String() string {
	return fmt.Sprintf("%s: %s %s", p.ProcessExec, p.ProcessName, p.ProcessArgs)
}

// NetworkConn represents a network connection.
// Fields are sorted by their size to optimize for memory padding.
type NetworkConn struct {
	SrcEntity networkgraph.Entity // ~38 bytes
	DstEntity networkgraph.Entity // ~38 bytes
	Protocol  storage.L4Protocol  // 4 bytes
	DstPort   uint16              // 2 bytes
}

func (i *NetworkConn) ToProto(ts timestamp.MicroTS) *storage.NetworkFlow {
	proto := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  i.SrcEntity.ToProto(),
			DstEntity:  i.DstEntity.ToProto(),
			DstPort:    uint32(i.DstPort),
			L4Protocol: i.Protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastSeenTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}
	return proto
}

func (i *NetworkConn) Key(h hash.Hash64) string {
	return i.keyHash(h)
}

// BinaryKey generates a binary hash for memory-efficient storage in dedupers
func (i *NetworkConn) BinaryKey() BinaryHash {
	return i.binaryKeyHash()
}

// ContainerEndpoint is a key in Sensor's maps that track active endpoints. It's set of fields should be minimal.
// Fields are sorted by their size to optimize for memory padding.
type ContainerEndpoint struct {
	Entity   networkgraph.Entity // ~38 bytes
	Protocol storage.L4Protocol  // 4 bytes
	Port     uint16              // 2 bytes
}

func (i *ContainerEndpoint) ToProto(ts timestamp.MicroTS) *storage.NetworkEndpoint {
	proto := &storage.NetworkEndpoint{
		Props: &storage.NetworkEndpointProperties{
			Entity:     i.Entity.ToProto(),
			Port:       uint32(i.Port),
			L4Protocol: i.Protocol,
		},
	}

	if ts != timestamp.InfiniteFuture {
		proto.LastActiveTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}
	return proto
}

// BinaryKey generates a binary hash for memory-efficient storage in dedupers
func (i *ContainerEndpoint) BinaryKey(h hash.Hash64) BinaryHash {
	return i.binaryKeyHash(h)
}

// ProcessListening represents a listening process.
// Fields are sorted by their size to optimize for memory padding.
// Note that ProcessListening is a composition of fields from two sources:
// `containerEndpoint` and `clusterentities.ContainerMetadata`.
// This struct in enriched only when new `containerEndpoint` data arrives.
type ProcessListening struct {
	Process       ProcessInfo        // 48 bytes (3 strings)
	PodID         string             // 16 bytes
	ContainerName string             // 16 bytes
	DeploymentID  string             // 16 bytes
	PodUID        string             // 16 bytes
	Namespace     string             // 16 bytes
	Protocol      storage.L4Protocol // 4 bytes
	Port          uint16             // 2 bytes
}

type ProcessListeningWithTimestamp struct {
	ProcessListening *ProcessListening
	LastSeen         timestamp.MicroTS
}

func (i *ProcessListening) ToProto(ts timestamp.MicroTS) *storage.ProcessListeningOnPortFromSensor {
	proto := &storage.ProcessListeningOnPortFromSensor{
		Port:     uint32(i.Port),
		Protocol: i.Protocol,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               i.PodID,
			ContainerName:       i.ContainerName,
			ProcessName:         i.Process.ProcessName,
			ProcessExecFilePath: i.Process.ProcessExec,
			ProcessArgs:         i.Process.ProcessArgs,
		},
		DeploymentId: i.DeploymentID,
		PodUid:       i.PodUID,
		Namespace:    i.Namespace,
	}

	if ts != timestamp.InfiniteFuture {
		proto.CloseTimestamp = protoconv.ConvertMicroTSToProtobufTS(ts)
	}

	return proto
}

// BinaryKey generates a binary hash for memory-efficient storage in dedupers
func (i *ProcessListening) BinaryKey(h hash.Hash64) BinaryHash {
	return i.binaryKeyHash(h)
}
