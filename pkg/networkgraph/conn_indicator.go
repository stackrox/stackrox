package networkgraph

import "github.com/stackrox/rox/generated/storage"

// NetworkConnIndicator provides a medium to uniquely identify network connections.
type NetworkConnIndicator struct {
	srcEntity Entity
	dstEntity Entity
	dstPort   uint32
	protocol  storage.L4Protocol
}

// GetNetworkConnIndicator constructs an indicator for supplied network connection.
func GetNetworkConnIndicator(conn *storage.NetworkFlow) NetworkConnIndicator {
	return NetworkConnIndicator{
		srcEntity: EntityFromProto(conn.GetProps().GetSrcEntity()),
		dstEntity: EntityFromProto(conn.GetProps().GetDstEntity()),
		protocol:  conn.GetProps().GetL4Protocol(),
		dstPort:   conn.GetProps().GetDstPort(),
	}
}
