package aggregator

import (
	"errors"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// NetworkConnsAggregator provides functionality to aggregate supplied network connections into a new slice.
//go:generate mockgen-wrapper
type NetworkConnsAggregator interface {
	Aggregate(conns []*storage.NetworkFlow) []*storage.NetworkFlow
}

// NewSubnetToSupernetConnAggregator returns a NetworkConnsAggregator that aggregates all network connections into
// immediate supernet. Atleast one network tree must be specified.
func NewSubnetToSupernetConnAggregator(trees ...tree.ReadOnlyNetworkTree) (NetworkConnsAggregator, error) {
	if len(trees) == 0 {
		return nil, errors.New("at least one network tree must be provided")
	}

	return &aggregateToSupernetImpl{
		trees: trees,
	}, nil
}

// NewDefaultToCustomExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates all network connections with default
// network connections into immediate non-default (custom) supernet.
func NewDefaultToCustomExtSrcConnAggregator(networkTree tree.ReadOnlyNetworkTree) (NetworkConnsAggregator, error) {
	if networkTree == nil {
		return nil, errors.New("network tree must be provided")
	}

	return &aggregateDefaultToCustomExtSrcsImpl{
		networkTree: networkTree,
	}, nil
}

// NewDuplicateNameExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates multiple external network
// connections with same external endpoint, as determined by name, into a single connection.
func NewDuplicateNameExtSrcConnAggregator() NetworkConnsAggregator {
	return &aggregateExternalConnByNameImpl{}
}
