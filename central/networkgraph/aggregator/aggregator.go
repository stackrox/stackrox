package aggregator

import (
	"errors"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// NetworkConnsAggregator provides functionality to aggregate supplied network connections into a new slice.
//
//go:generate mockgen-wrapper
type NetworkConnsAggregator interface {
	Aggregate(conns []*storage.NetworkFlow) []*storage.NetworkFlow
}

// NewSubnetToSupernetConnAggregator returns a NetworkConnsAggregator that aggregates network connections whose
// external endpoints are not found in the network tree into conns with supernet endpoints. At least one network tree
// must be specified.
func NewSubnetToSupernetConnAggregator(networkTree tree.ReadOnlyNetworkTree) (NetworkConnsAggregator, error) {
	if networkTree == nil {
		return nil, errors.New("network tree must be provided")
	}

	return &aggregateToSupernetImpl{
		tree:         networkTree,
		supernetPred: func(e *storage.NetworkEntityInfo) bool { return true },
	}, nil
}

// NewDefaultToCustomExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates all network connections with default
// network connections into immediate non-default (custom) supernet.
func NewDefaultToCustomExtSrcConnAggregator(networkTree tree.ReadOnlyNetworkTree) (NetworkConnsAggregator, error) {
	if networkTree == nil {
		return nil, errors.New("network tree must be provided")
	}

	return &aggregateDefaultToCustomExtSrcsImpl{
		networkTree:  networkTree,
		supernetPred: func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() },
	}, nil
}

// NewDuplicateNameExtSrcConnAggregator returns a NetworkConnsAggregator that aggregates multiple external network
// connections with same external endpoint, as determined by name, into a single connection.
func NewDuplicateNameExtSrcConnAggregator() NetworkConnsAggregator {
	return &aggregateExternalConnByNameImpl{}
}
