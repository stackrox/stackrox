package indicator

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

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

// Key produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two NetworkConn's are identical (for the network-graph purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster, but the resulting string takes more memory than for HashedKey.
func (i *NetworkConn) Key() string {
	var buf strings.Builder
	// 82 chars is an estimate based on typical string-lengths of the NetworkConn's fields to avoid re-sizing.
	// 3 chars of the delimiters can be saved, but would only reduce number of bytes allocated locally and
	// won't reduce the size of a large collection holding many NetworkConn's.
	buf.Grow(82)
	_, _ = buf.WriteString(i.SrcEntity.ID)                             // 36 chars for UUIDv4
	_ = buf.WriteByte(':')                                             // 1 char for optional delimiter
	_, _ = buf.WriteString(i.DstEntity.ID)                             // 36 chars for UUIDv4
	_ = buf.WriteByte(':')                                             // 1 char for optional delimiter
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.DstPort), 10))  // 5 chars maximally
	_ = buf.WriteByte(':')                                             // 1 char for optional delimiter
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10)) // 2 chars for the underlying enum-int (with sign)

	return buf.String()
}

// HashedKey produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two NetworkConn's are identical (for the network-graph purposes) when their keys are identical.
// This is memory-optimized implementation that is slower, but the resulting string takes less memory than for Key.
func (i *NetworkConn) HashedKey() string {
	h := fnv.New64a()
	// For a collection of length 10^N, the 64bit FNV-1a hash has approximate collision probability of 2.71x10^(N-4).
	// For example: for 100M uniformly distributed items, the collision probability is 2.71x10^4 = 0.027.
	// For lower collision probabilities, one needs to use a fast 128bit hash, for example: XXH3_128 (LLM recommendation).
	_, _ = h.Write([]byte(i.SrcEntity.ID))
	_, _ = h.Write([]byte(i.DstEntity.ID))
	// Hash the destination port (as bytes for efficiency)
	portBytes := [2]byte{byte(i.DstPort >> 8), byte(i.DstPort)}
	_, _ = h.Write(portBytes[:])
	// Hash the protocol (as bytes for efficiency)
	protocolBytes := [4]byte{
		byte(i.Protocol >> 24), byte(i.Protocol >> 16),
		byte(i.Protocol >> 8), byte(i.Protocol),
	}
	_, _ = h.Write(protocolBytes[:])
	// Return as 16-character hex string (8 bytes * 2 hex chars per byte)
	hash := h.Sum64()
	return hex.EncodeToString([]byte{
		byte(hash >> 56), byte(hash >> 48), byte(hash >> 40), byte(hash >> 32),
		byte(hash >> 24), byte(hash >> 16), byte(hash >> 8), byte(hash),
	})
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

// Key produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two ContainerEndpoint's are identical (for the network-graph purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster, but the resulting string takes more memory than for HashedKey.
func (i *ContainerEndpoint) Key() string {
	var buf strings.Builder
	buf.Grow(45) // Estimate based on typical ID lengths.

	_, _ = buf.WriteString(i.Entity.ID) // 36 chars (UUIDv4)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Port), 10)) // 5 chars
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10)) // 2 chars

	return buf.String()
}

// HashedKey produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two ContainerEndpoint's are identical (for the network-graph purposes) when their keys are identical.
// This is memory-optimized implementation that is slower, but the resulting string takes less memory than for Key.
func (i *ContainerEndpoint) HashedKey() string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(i.Entity.ID))
	// Hash the destination port (as bytes for efficiency)
	portBytes := [2]byte{byte(i.Port >> 8), byte(i.Port)}
	_, _ = h.Write(portBytes[:])
	// Hash the protocol (as bytes for efficiency)
	protocolBytes := [4]byte{
		byte(i.Protocol >> 24), byte(i.Protocol >> 16),
		byte(i.Protocol >> 8), byte(i.Protocol),
	}
	_, _ = h.Write(protocolBytes[:])
	// Return as 16-character hex string (8 bytes * 2 hex chars per byte)
	hash := h.Sum64()
	return hex.EncodeToString([]byte{
		byte(hash >> 56), byte(hash >> 48), byte(hash >> 40), byte(hash >> 32),
		byte(hash >> 24), byte(hash >> 16), byte(hash >> 8), byte(hash),
	})
}

// ProcessListening represents a listening process.
// Fields are sorted by their size to optimize for memory padding.
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

// Key produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two ProcessListening's are identical (for the network-graph & PLoP purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster, but the resulting string takes more memory than for HashedKey.
func (i *ProcessListening) Key() string {
	var buf strings.Builder
	// It is hard to compute any reasonable size for pre-allocation as many items have variable length.
	// Estimating partially based on gut feeling.
	buf.Grow(170)

	// Skipping some fields to save memory - they should not be required to ensure uniqueness.
	_, _ = buf.WriteString(i.PodID) // This is a pod name - variable, assume 30
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.Process.ProcessName) // variable len, assume 30
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.Process.ProcessExec) // variable len, assume 30
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.Process.ProcessArgs) // variable len, assume 30
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Port), 10)) // 5 chars
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10)) // 2 chars
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.PodUID) // 36 chars (UUIDv4)

	return buf.String()
}

// HashedKey produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two ProcessListening's are identical (for the network-graph & PLoP purposes) when their keys are identical.
// This is memory-optimized implementation that is slower, but the resulting string takes less memory than for Key.
func (i *ProcessListening) HashedKey() string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(i.PodID))
	_, _ = h.Write([]byte(i.Process.ProcessName))
	_, _ = h.Write([]byte(i.Process.ProcessExec))
	_, _ = h.Write([]byte(i.Process.ProcessArgs))
	_, _ = h.Write([]byte(i.PodUID))
	// Hash the destination port (as bytes for efficiency)
	portBytes := [2]byte{byte(i.Port >> 8), byte(i.Port)}
	_, _ = h.Write(portBytes[:])
	// Hash the protocol (as bytes for efficiency)
	protocolBytes := [4]byte{
		byte(i.Protocol >> 24), byte(i.Protocol >> 16),
		byte(i.Protocol >> 8), byte(i.Protocol),
	}
	_, _ = h.Write(protocolBytes[:])
	// Return as 16-character hex string (8 bytes * 2 hex chars per byte)
	hash := h.Sum64()
	return hex.EncodeToString([]byte{
		byte(hash >> 56), byte(hash >> 48), byte(hash >> 40), byte(hash >> 32),
		byte(hash >> 24), byte(hash >> 16), byte(hash >> 8), byte(hash),
	})
}
