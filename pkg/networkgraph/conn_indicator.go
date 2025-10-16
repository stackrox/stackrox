package networkgraph

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// NetworkConnIndicator provides a medium to uniquely identify network connections.
type NetworkConnIndicator struct {
	SrcEntity Entity
	DstEntity Entity
	DstPort   uint32
	Protocol  storage.L4Protocol
}

// GetNetworkConnIndicator constructs an indicator for supplied network connection.
func GetNetworkConnIndicator(conn *storage.NetworkFlow) NetworkConnIndicator {
	return NetworkConnIndicator{
		SrcEntity: EntityFromProto(conn.GetProps().GetSrcEntity()),
		DstEntity: EntityFromProto(conn.GetProps().GetDstEntity()),
		Protocol:  conn.GetProps().GetL4Protocol(),
		DstPort:   conn.GetProps().GetDstPort(),
	}
}

// String returns the string representation of NetworkConnIndicator.
func (i NetworkConnIndicator) String() string {
	return fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(i.SrcEntity.Type), i.SrcEntity.ID, int32(i.DstEntity.Type), i.DstEntity.ID, i.DstPort, int32(i.Protocol))
}

// ToNetworkFlowPropertiesProto converts the proto to a network flow properties.
func (i *NetworkConnIndicator) ToNetworkFlowPropertiesProto() *storage.NetworkFlowProperties {
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(i.SrcEntity.ToProto())
	nfp.SetDstEntity(i.DstEntity.ToProto())
	nfp.SetDstPort(i.DstPort)
	nfp.SetL4Protocol(i.Protocol)
	return nfp

}
