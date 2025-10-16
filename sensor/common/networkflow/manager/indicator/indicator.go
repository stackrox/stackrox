package indicator

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

// HashingAlgo selects the algorithm for hashing the connection/endpoint/process fingerprinting.
type HashingAlgo int

const (
	HashingAlgoString HashingAlgo = iota
	HashingAlgoHash
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
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(i.SrcEntity.ToProto())
	nfp.SetDstEntity(i.DstEntity.ToProto())
	nfp.SetDstPort(uint32(i.DstPort))
	nfp.SetL4Protocol(i.Protocol)
	proto := &storage.NetworkFlow{}
	proto.SetProps(nfp)

	if ts != timestamp.InfiniteFuture {
		proto.SetLastSeenTimestamp(protoconv.ConvertMicroTSToProtobufTS(ts))
	}
	return proto
}

func (i *NetworkConn) Key(h HashingAlgo) string {
	switch h {
	case HashingAlgoString:
		return i.keyString()
	default:
		return i.keyHash()
	}
}

// ContainerEndpoint is a key in Sensor's maps that track active endpoints. It's set of fields should be minimal.
// Fields are sorted by their size to optimize for memory padding.
type ContainerEndpoint struct {
	Entity   networkgraph.Entity // ~38 bytes
	Protocol storage.L4Protocol  // 4 bytes
	Port     uint16              // 2 bytes
}

func (i *ContainerEndpoint) ToProto(ts timestamp.MicroTS) *storage.NetworkEndpoint {
	nep := &storage.NetworkEndpointProperties{}
	nep.SetEntity(i.Entity.ToProto())
	nep.SetPort(uint32(i.Port))
	nep.SetL4Protocol(i.Protocol)
	proto := &storage.NetworkEndpoint{}
	proto.SetProps(nep)

	if ts != timestamp.InfiniteFuture {
		proto.SetLastActiveTimestamp(protoconv.ConvertMicroTSToProtobufTS(ts))
	}
	return proto
}

func (i *ContainerEndpoint) Key(h HashingAlgo) string {
	switch h {
	case HashingAlgoString:
		return i.keyString()
	default:
		return i.keyHash()
	}
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
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(i.PodID)
	piuk.SetContainerName(i.ContainerName)
	piuk.SetProcessName(i.Process.ProcessName)
	piuk.SetProcessExecFilePath(i.Process.ProcessExec)
	piuk.SetProcessArgs(i.Process.ProcessArgs)
	proto := &storage.ProcessListeningOnPortFromSensor{}
	proto.SetPort(uint32(i.Port))
	proto.SetProtocol(i.Protocol)
	proto.SetProcess(piuk)
	proto.SetDeploymentId(i.DeploymentID)
	proto.SetPodUid(i.PodUID)
	proto.SetNamespace(i.Namespace)

	if ts != timestamp.InfiniteFuture {
		proto.SetCloseTimestamp(protoconv.ConvertMicroTSToProtobufTS(ts))
	}

	return proto
}

func (i *ProcessListening) Key(h HashingAlgo) string {
	switch h {
	case HashingAlgoString:
		return i.keyString()
	default:
		return i.keyHash()
	}
}
