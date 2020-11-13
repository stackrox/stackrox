package tree

import (
	"github.com/stackrox/rox/generated/storage"
)

// NetworkTree provides functionality to store network entities per supernet-subnet relationship.
type NetworkTree interface {
	ReadOnlyNetworkTree

	Insert(entity *storage.NetworkEntityInfo) error
	Remove(key string)
}

// ReadOnlyNetworkTree provides functionality to read network entities from a network tree.
type ReadOnlyNetworkTree interface {
	Cardinality() int
	GetSupernet(key string) *storage.NetworkEntityInfo
	GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo
	GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo
	GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo
	GetSubnets(key string) []*storage.NetworkEntityInfo
	GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo
	Get(key string) *storage.NetworkEntityInfo
	Search(key string) bool
}
