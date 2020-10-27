package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

type defaultToCustomExtSrcAggregator struct {
	networkTree *tree.NetworkTreeWrapper
}

// AggregateExternalConnections aggregates multiple external network connections with same external endpoint,
// as determined by name, into a single connection.
func (a *defaultToCustomExtSrcAggregator) Aggregate(conns []*storage.NetworkFlow) []*storage.NetworkFlow {
	normalizedConns := make(map[networkgraph.NetworkConnIndicator]*storage.NetworkFlow)
	ret := make([]*storage.NetworkFlow, 0, len(conns)/4)
	supernetCache := make(map[string]*storage.NetworkEntityInfo)

	for _, conn := range conns {
		srcEntity, dstEntity := conn.GetProps().GetSrcEntity(), conn.GetProps().GetDstEntity()
		// This is essentially an invalid connection.
		if srcEntity == nil || dstEntity == nil {
			continue
		}

		// If both endpoints are not external (including INTERNET), skip processing.
		if !networkgraph.IsExternal(srcEntity) && !networkgraph.IsExternal(dstEntity) {
			ret = append(ret, conn)
			continue
		}

		// Move the connection from default external network to non-default supernet. If none is found, it gets mapped to INTERNET.
		if networkgraph.IsKnownDefaultExternal(srcEntity) {
			conn.Props.SrcEntity = a.getSupernet(srcEntity.GetId(), supernetCache)
		} else if networkgraph.IsKnownDefaultExternal(dstEntity) {
			conn.Props.DstEntity = a.getSupernet(dstEntity.GetId(), supernetCache)
		}

		connID := networkgraph.GetNetworkConnIndicator(conn)
		if storedFlow := normalizedConns[connID]; storedFlow != nil {
			if storedFlow.GetLastSeenTimestamp().Compare(conn.GetLastSeenTimestamp()) < 0 {
				storedFlow.LastSeenTimestamp = conn.GetLastSeenTimestamp()
			}
		} else {
			normalizedConns[connID] = conn
		}
	}

	for _, conn := range normalizedConns {
		ret = append(ret, conn)
	}
	return ret
}

func (a *defaultToCustomExtSrcAggregator) getSupernet(id string, cache map[string]*storage.NetworkEntityInfo) *storage.NetworkEntityInfo {
	supernet := cache[id]
	if supernet == nil {
		supernet = a.networkTree.GetMatchingSupernet(id, func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() })
		cache[id] = supernet
	}
	return supernet
}
