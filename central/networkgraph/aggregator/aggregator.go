package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// NetworkConnsAggregator provides functionality to aggregate supplied network connections into a new slice.
type NetworkConnsAggregator interface {
	Aggregate(conns []*storage.NetworkFlow) []*storage.NetworkFlow
}

// NewDefaultToCustomExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates all network connections with default
// network connections into immediate non-default (custom) supernet.
func NewDefaultToCustomExtSrcConnAggregator(networkTree tree.ReadOnlyNetworkTree) NetworkConnsAggregator {
	return &aggregateDefaultToCustomExtSrcsImpl{
		networkTree: networkTree,
	}
}

// NewDuplicateNameExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates multiple external network
// connections with same external endpoint, as determined by name, into a single connection.
func NewDuplicateNameExtSrcConnAggregator() NetworkConnsAggregator {
	return &aggregateExternalConnByNameImpl{}
}
