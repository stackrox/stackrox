package tree

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// NetworkTree provides functionality to store network entities per supernet-subnet relationship.
type NetworkTree interface {
	ReadOnlyNetworkTree

	Insert(entity *storage.NetworkEntityInfo) error
	Remove(key string)
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
