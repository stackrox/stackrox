package tree

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
)

// NetworkTreeWrapper is a wrapper around NetworkTree structure that allows dealing with network entities IPv4 as well as IPv6 address family.
type NetworkTreeWrapper struct {
	trees map[pkgNet.Family]*NetworkTree

	lock sync.RWMutex
}

// NewDefaultNetworkTreeWrapper returns a new instance of NetworkTreeWrapper.
func NewDefaultNetworkTreeWrapper() (*NetworkTreeWrapper, error) {
	ipv4Tree, err := NewDefaultNetworkTree(pkgNet.IPv4)
	if err != nil {
		return nil, err
	}

	ipv6Tree, err := NewDefaultNetworkTree(pkgNet.IPv6)
	if err != nil {
		return nil, err
	}

	trees := make(map[pkgNet.Family]*NetworkTree)
	trees[pkgNet.IPv4] = ipv4Tree
	trees[pkgNet.IPv6] = ipv6Tree

	return &NetworkTreeWrapper{
		trees: trees,
	}, nil
}

// NewNetworkTreeWrapper returns a new instance of NetworkTreeWrapper for the supplied list of network entities.
func NewNetworkTreeWrapper(entities []*storage.NetworkEntityInfo) (*NetworkTreeWrapper, error) {
	wrapper, err := NewDefaultNetworkTreeWrapper()
	if err != nil {
		return nil, err
	}

	if err := wrapper.build(entities); err != nil {
		return nil, err
	}
	return wrapper, nil
}

func (t *NetworkTreeWrapper) build(entities []*storage.NetworkEntityInfo) error {
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
			n := ipNet.AsIPNet()
			if err := t.trees[family].insertNoValidate(ipNetToEntity[ipNet], &net.IPNet{IP: n.IP, Mask: n.Mask}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Insert add the supplied network entity. If a entity with the same key is already present in a tree,
// the CIDR of stored entity is updated and the tree is rearranged to maintain the supernet-subnet relationship.
func (t *NetworkTreeWrapper) Insert(entity *storage.NetworkEntityInfo) error {
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
func (t *NetworkTreeWrapper) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			tree.Remove(key)
			break
		}
	}
}

// GetSupernet returns the direct supernet (predecessor) that fully contains the network for given key, if present.
func (t *NetworkTreeWrapper) GetSupernet(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return tree.GetSupernet(key)
		}
	}
	return nil
}

// GetMatchingSupernet returns the direct supernet (predecessor) that fully contains the network for given key and satisfies the predicate.
func (t *NetworkTreeWrapper) GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return tree.GetMatchingSupernet(key, pred)
		}
	}
	return nil
}

// GetSubnets returns all the direct subnets (successor) contained by the network for given key, if present.
func (t *NetworkTreeWrapper) GetSubnets(key string) []*storage.NetworkEntityInfo {
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

// Get returns the network entity for given key, if present, otherwise nil.
func (t *NetworkTreeWrapper) Get(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	entity, _ := t.getWithFamilyNoLock(key)
	return entity
}

func (t *NetworkTreeWrapper) getWithFamilyNoLock(key string) (*storage.NetworkEntityInfo, pkgNet.Family) {
	for family, tree := range t.trees {
		if tree.Search(key) {
			return tree.Get(key), family
		}
	}
	return nil, pkgNet.InvalidFamily
}

// Search return true if the network entity for the given key is found in the network trees.
func (t *NetworkTreeWrapper) Search(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Search(key) {
			return true
		}
	}
	return false
}
