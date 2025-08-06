package indicator

import (
	"fmt"
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

// NetworkConn represents a network connection for update computation
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

// Key returns a string representation of the network connection.
// Benchmarked for balance between cpu performance and memory usage.
func (i *NetworkConn) Key() string {
	var buf strings.Builder
	buf.Grow(100) // Estimate based on typical ID lengths

	_, _ = buf.WriteString(i.SrcEntity.ID)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.DstEntity.ID)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.DstPort), 10))
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10))

	return buf.String()
}

// ContainerEndpoint is a key in Sensor's maps that track active endpoints. It's set of fields should be minimal.
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

func (i *ContainerEndpoint) Key() string {
	var buf strings.Builder
	buf.Grow(100) // Estimate based on typical ID lengths // FIXME: re-estimate

	_, _ = buf.WriteString(i.Entity.ID)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Port), 10))
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10))

	return buf.String()
}

// ProcessListening represents a listening process for update computation
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

func (i *ProcessListening) Key() string {
	var buf strings.Builder
	buf.Grow(100) // Estimate based on typical ID lengths // FIXME: re-estimate

	_, _ = buf.WriteString(i.PodID)
	_ = buf.WriteByte(':')
	//buf.WriteString(i.ContainerName)
	//buf.WriteByte(':')
	//buf.WriteString(i.DeploymentID)
	//buf.WriteByte(':')
	_, _ = buf.WriteString(i.Process.ProcessName)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.Process.ProcessExec)
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Port), 10))
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(strconv.FormatUint(uint64(i.Protocol), 10))
	_ = buf.WriteByte(':')
	_, _ = buf.WriteString(i.PodUID)
	//buf.WriteByte(':')
	//buf.WriteString(i.Namespace)

	return buf.String()
}
