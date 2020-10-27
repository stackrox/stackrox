package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// NetworkConnsAggregator provides functionality to aggregate supplied network connections into a new slice.
type NetworkConnsAggregator interface {
	Aggregate(conns []*storage.NetworkFlow) []*storage.NetworkFlow
}

// NewDefaultToCustomExtSrcAggregator returns NetworkConnsAggregator that aggregates all network connections with default
// network connections into immediate non-default (custom) supernet.
func NewDefaultToCustomExtSrcAggregator(networkTree *tree.NetworkTreeWrapper) NetworkConnsAggregator {
	return &defaultToCustomExtSrcAggregator{
		networkTree: networkTree,
	}
}
