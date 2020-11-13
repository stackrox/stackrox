package tree

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
)

// networkTreeWrapper is a wrapper around networkTreeImpl structure that allows dealing with network entities IPv4 as well as IPv6 address family.
type networkTreeWrapper struct {
	trees map[pkgNet.Family]NetworkTree

	lock sync.RWMutex
}

// NewDefaultNetworkTreeWrapper returns a new instance of networkTreeWrapper.
func NewDefaultNetworkTreeWrapper() NetworkTree {
	return newDefaultNetworkTreeWrapper()
}

// NewNetworkTreeWrapper returns a new instance of networkTreeWrapper for the supplied list of network entities.
func NewNetworkTreeWrapper(entities []*storage.NetworkEntityInfo) (NetworkTree, error) {
	wrapper := newDefaultNetworkTreeWrapper()
	if err := wrapper.build(entities); err != nil {
		return nil, err
	}
	return wrapper, nil
}

func newDefaultNetworkTreeWrapper() *networkTreeWrapper {
	trees := make(map[pkgNet.Family]NetworkTree)
	trees[pkgNet.IPv4] = newDefaultNetworkTree(pkgNet.IPv4)
	trees[pkgNet.IPv6] = newDefaultNetworkTree(pkgNet.IPv6)

	return &networkTreeWrapper{
		trees: trees,
	}
}

func (t *networkTreeWrapper) build(entities []*storage.NetworkEntityInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	netSliceByFamily := make(map[pkgNet.Family][]pkgNet.IPNetwork)
	ipNetToEntity := make(map[pkgNet.IPNetwork]*storage.NetworkEntityInfo)

	for _, entity := range entities {
		ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
		if !ipNet.IsValid() {
			return errors.Errorf("received invalid CIDR %s to insert", entity.GetExternalSource().GetCidr())
		}

		ipNetToEntity[ipNet] = entity
		netSliceByFamily[ipNet.Family()] = append(netSliceByFamily[ipNet.Family()], ipNet)
	}

	// Sort the network by prefix length to reduce the tree re-arrangement.
	for family, netSlice := range netSliceByFamily {
		normalizeNetworks(family, netSlice)
	}

	for family, netSlice := range netSliceByFamily {
		for _, ipNet := range netSlice {
			if err := t.trees[family].Insert(ipNetToEntity[ipNet]); err != nil {
				return err
			}
		}
	}
	return nil
}

// Cardinality returns the number of networks in the tree.
func (t *networkTreeWrapper) Cardinality() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	ret := 0
	for _, t := range t.trees {
		ret += t.Cardinality()
	}
	return ret
}

// Insert add the supplied network entity. If a entity with the same key is already present in a tree,
// the CIDR of stored entity is updated and the tree is rearranged to maintain the supernet-subnet relationship.
func (t *networkTreeWrapper) Insert(entity *storage.NetworkEntityInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
	if !ipNet.IsValid() {
		return errors.Errorf("received invalid CIDR %s to insert", entity.GetExternalSource().GetCidr())
	}

	if storedEntity, family := t.getWithFamilyNoLock(entity.GetId()); storedEntity != nil && family != ipNet.Family() {
		t.trees[family].Remove(entity.GetId())
	}

	return t.trees[ipNet.Family()].Insert(entity)
}

// Remove removes the network entity from a tree for given key, if present.
func (t *networkTreeWrapper) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			tree.Remove(key)
			break
		}
	}
}

// GetSupernet returns the smallest supernet that fully contains the network for given key, if present.
func (t *networkTreeWrapper) GetSupernet(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return tree.GetSupernet(key)
		}
	}
	return nil
}

// GetMatchingSupernet returns the smallest supernet that fully contains the network for given key and satisfies the predicate.
func (t *networkTreeWrapper) GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return tree.GetMatchingSupernet(key, pred)
		}
	}
	return nil
}

// GetSupernetForCIDR returns the smallest supernet that fully contains the network for the given CIDR.
func (t *networkTreeWrapper) GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo {
	ipNet := pkgNet.IPNetworkFromCIDR(cidr)
	if !ipNet.IsValid() {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.trees[ipNet.Family()].GetSupernetForCIDR(cidr)
}

// GetMatchingSupernetForCIDR returns the smallest supernet that fully contains the supplied network and satisfies the predicate.
func (t *networkTreeWrapper) GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	ipNet := pkgNet.IPNetworkFromCIDR(cidr)
	if !ipNet.IsValid() {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.trees[ipNet.Family()].GetMatchingSupernetForCIDR(cidr, supernetPred)
}

// GetSubnets returns the largest disjoint subnets contained by the network for given key, if present.
func (t *networkTreeWrapper) GetSubnets(key string) []*storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// The subnet of INTERNET lies in both the trees.
	if key == networkgraph.InternetExternalSourceID {
		var ret []*storage.NetworkEntityInfo
		for _, tree := range t.trees {
			if tree.Search(key) {
				ret = append(ret, tree.GetSubnets(key)...)
			}
		}
		return ret
	}

	for _, tree := range t.trees {
		if tree.Search(key) {
			return tree.GetSubnets(key)
		}
	}
	return nil
}

// GetSubnetsForCIDR returns the largest disjoint subnets contained by the given network, if any.
func (t *networkTreeWrapper) GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo {
	ipNet := pkgNet.IPNetworkFromCIDR(cidr)
	if !ipNet.IsValid() {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.trees[ipNet.Family()].GetSubnetsForCIDR(cidr)
}

// Get returns the network entity for given key, if present, otherwise nil.
func (t *networkTreeWrapper) Get(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	entity, _ := t.getWithFamilyNoLock(key)
	return entity
}

func (t *networkTreeWrapper) getWithFamilyNoLock(key string) (*storage.NetworkEntityInfo, pkgNet.Family) {
	for family, tree := range t.trees {
		if tree.Search(key) {
			return tree.Get(key), family
		}
	}
	return nil, pkgNet.InvalidFamily
}

// Search return true if the network entity for the given key is found in the network trees.
func (t *networkTreeWrapper) Search(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return true
		}
	}
	return false
}
