package tree

import (
	"github.com/stackrox/rox/generated/storage"
)

// NetworkTree provides functionality to store network entities per supernet-subnet relationship.
type NetworkTree interface {
	ReadOnlyNetworkTree

	Insert(entity *storage.NetworkEntityInfo) error
	Remove(key string)
	// Checks that there are no leafs without values, that the number of values is equal to
	// the cardinality, and that the values in the network tree corresponds to the paths needed
	// to take from the root to their location. If there are multiple trees, the checks are done
	// for each tree.
	ValidateNetworkTree() bool
}

// ReadOnlyNetworkTree provides functionality to read network entities from a network tree.
type ReadOnlyNetworkTree interface {
	// Returns the number of networks in the tree.
	Cardinality() int
	// Returns the smallest subnet larger than, and that fully contains the network of queried key.
	GetSupernet(key string) *storage.NetworkEntityInfo
	// Returns the smallest subnet larger than, and that fully contains the network of queried key, that matches the predicate.
	GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo
	// Returns the smallest subnet larger than, and that fully contains the queried network.
	GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo
	// Returns the smallest subnet larger than, and that fully contains the queried network, that matches the predicate.
	GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo
	// Returns the largest networks smaller than, and fully contained by the network of queried key.
	GetSubnets(key string) []*storage.NetworkEntityInfo
	// Returns the largest networks smaller than, and fully contained by the queried network.
	GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo
	Get(key string) *storage.NetworkEntityInfo
	Exists(key string) bool
}
