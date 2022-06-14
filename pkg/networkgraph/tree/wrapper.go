package tree

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
)

// networkTreeWrapper is a wrapper around networkTreeImpl structure that handles both IPv4 and IPv6 networks.
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
	entitiesByAddrFamily := make(map[pkgNet.Family][]*storage.NetworkEntityInfo)
	for _, entity := range entities {
		ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
		if !ipNet.IsValid() {
			return nil, errors.Errorf("received invalid CIDR %s to insert", entity.GetExternalSource().GetCidr())
		}
		entitiesByAddrFamily[ipNet.Family()] = append(entitiesByAddrFamily[ipNet.Family()], entity)
	}

	trees := make(map[pkgNet.Family]NetworkTree)
	tree, err := NewNRadixTree(pkgNet.IPv4, entitiesByAddrFamily[pkgNet.IPv4])
	if err != nil {
		return nil, err
	}
	trees[pkgNet.IPv4] = tree

	tree, err = NewNRadixTree(pkgNet.IPv6, entitiesByAddrFamily[pkgNet.IPv6])
	if err != nil {
		return nil, err
	}
	trees[pkgNet.IPv6] = tree

	return &networkTreeWrapper{
		trees: trees,
	}, nil
}

func newDefaultNetworkTreeWrapper() *networkTreeWrapper {
	return &networkTreeWrapper{
		trees: map[pkgNet.Family]NetworkTree{
			pkgNet.IPv4: NewDefaultNRadixTree(pkgNet.IPv4),
			pkgNet.IPv6: NewDefaultNRadixTree(pkgNet.IPv6),
		},
	}
}

// Cardinality returns the number of networks in the tree.
func (t *networkTreeWrapper) Cardinality() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// Since we have IPv4 root and IPv6 root.
	ret := -1
	for _, t := range t.trees {
		ret += t.Cardinality()
	}
	return ret
}

// Insert add the supplied network entity. Values for existing conflicting keys are overwritten.
func (t *networkTreeWrapper) Insert(entity *storage.NetworkEntityInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	ipNet := pkgNet.IPNetworkFromCIDR(entity.GetExternalSource().GetCidr())
	if !ipNet.IsValid() {
		return errors.Errorf("received invalid CIDR %s to insert", entity.GetExternalSource().GetCidr())
	}

	if storedEntity, family := t.getWithFamilyNoLock(entity.GetId()); storedEntity != nil {
		t.trees[family].Remove(entity.GetId())
	}

	return t.trees[ipNet.Family()].Insert(entity)
}

// Remove removes the network entity from a tree for given key, if present.
func (t *networkTreeWrapper) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, tree := range t.trees {
		if tree.Exists(key) {
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
		if tree.Exists(key) {
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
		if tree.Exists(key) {
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
			if nets := tree.GetSubnets(key); len(nets) != 0 {
				ret = append(ret, nets...)
			}
		}
		return ret
	}

	for _, tree := range t.trees {
		if tree.Exists(key) {
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
		if val := tree.Get(key); val != nil {
			return val, family
		}
	}
	return nil, pkgNet.InvalidFamily
}

// Search return true if the network entity for the given key is found in the network trees.
func (t *networkTreeWrapper) Exists(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Exists(key) {
			return true
		}
	}
	return false
}
