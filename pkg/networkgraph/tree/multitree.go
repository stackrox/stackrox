package tree

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	pkgNet "github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

// multiReadOnlyNTree is a wrapper around networkTreeWrapper that handles multiples trees.
type multiReadOnlyNTree struct {
	trees []ReadOnlyNetworkTree

	lock sync.RWMutex
}

// NewMultiNetworkTree returns a new instance of multiReadOnlyNTree for the supplied list of network trees.
func NewMultiNetworkTree(trees ...ReadOnlyNetworkTree) ReadOnlyNetworkTree {
	filtered := make([]ReadOnlyNetworkTree, 0, len(trees))
	for _, t := range trees {
		if t != nil {
			filtered = append(filtered, t)
		}
	}

	// If no valid tree is supplied create a default tree which contains INTERNET node only.
	if len(filtered) == 0 {
		filtered = append(filtered, NewDefaultNetworkTreeWrapper())
	}

	return &multiReadOnlyNTree{
		trees: filtered,
	}
}

// Cardinality returns the number of networks in all the tree.
func (t *multiReadOnlyNTree) Cardinality() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	ret := 0
	for _, t := range t.trees {
		ret += t.Cardinality()
	}
	// Remove all the extra INTERNETS counted. Duplicate networks across trees are still included.
	return ret - len(t.trees) + 1
}

// GetSupernet returns the smallest supernet that fully contains the network for given key, if present.
func (t *multiReadOnlyNTree) GetSupernet(key string) *storage.NetworkEntityInfo {
	if match := t.GetMatchingSupernet(key, func(entity *storage.NetworkEntityInfo) bool { return true }); match != nil {
		return match
	}
	// Internet is supernet for everything.
	return networkgraph.InternetEntity().ToProto()
}

// GetMatchingSupernet returns the smallest supernet that fully contains the network for given key and satisfies the predicate.
func (t *multiReadOnlyNTree) GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	if entity := t.Get(key); entity != nil {
		return t.GetMatchingSupernetForCIDR(entity.GetExternalSource().GetCidr(), pred)
	}
	// Since the supernet has to satisfy the predicate, we do not return internet.
	return nil
}

// GetSupernetForCIDR returns the smallest supernet that fully contains the network for the given CIDR.
func (t *multiReadOnlyNTree) GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo {
	if match := t.GetMatchingSupernetForCIDR(cidr, func(entity *storage.NetworkEntityInfo) bool { return true }); match != nil {
		return match
	}
	return networkgraph.InternetEntity().ToProto()
}

// GetMatchingSupernetForCIDR returns the smallest supernet that fully contains the supplied network and satisfies the predicate.
func (t *multiReadOnlyNTree) GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	ipNet := pkgNet.IPNetworkFromCIDR(cidr)
	if !ipNet.IsValid() {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	var supernets []*storage.NetworkEntityInfo
	for _, tree := range t.trees {
		if match := tree.GetMatchingSupernetForCIDR(cidr, supernetPred); match != nil {
			supernets = append(supernets, match)
		}
	}

	return getSmallestSupernet(supernets...)
}

// GetSubnets returns the largest disjoint subnets contained by the network for given key, if present.
func (t *multiReadOnlyNTree) GetSubnets(key string) []*storage.NetworkEntityInfo {
	entity := t.Get(key)
	if entity == nil {
		return nil
	}

	if entity.GetId() != networkgraph.InternetExternalSourceID {
		t.GetSubnetsForCIDR(entity.GetExternalSource().GetCidr())
	}

	if entity.GetId() == networkgraph.InternetExternalSourceID {
		ipv4s := getLargestSubnets(t.GetSubnetsForCIDR(ipv4InternetCIDR)...)
		ipv6s := getLargestSubnets(t.GetSubnetsForCIDR(ipv6InternetCIDR)...)

		ret := make([]*storage.NetworkEntityInfo, 0, len(ipv4s)+len(ipv6s))
		ret = append(ret, ipv4s...)
		ret = append(ret, ipv6s...)
		return ret
	}
	return t.GetSubnetsForCIDR(entity.GetExternalSource().GetCidr())

}

// GetSubnetsForCIDR returns the largest disjoint subnets contained by the given network, if any.
func (t *multiReadOnlyNTree) GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var subnets []*storage.NetworkEntityInfo
	for _, tree := range t.trees {
		if matches := tree.GetSubnetsForCIDR(cidr); matches != nil {
			subnets = append(subnets, matches...)
		}
	}
	return getLargestSubnets(subnets...)
}

// Get returns the network entity for given key, if present, otherwise nil.
func (t *multiReadOnlyNTree) Get(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if val := tree.Get(key); val != nil {
			return val
		}
	}
	return nil
}

// Search return true if the network entity for the given key is found in the network trees.
func (t *multiReadOnlyNTree) Exists(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, tree := range t.trees {
		if tree.Exists(key) {
			return true
		}
	}
	return false
}

func getSmallestSupernet(entities ...*storage.NetworkEntityInfo) *storage.NetworkEntityInfo {
	ret := networkgraph.InternetEntity().ToProto()
	largestPrefixSoFar := 0
	for _, entity := range entities {
		if entity == nil {
			continue
		}

		// Special case since Internet entity does not have CIDR.
		if entity.GetId() == networkgraph.InternetExternalSourceID {
			continue
		}

		_, ipNet, err := net.ParseCIDR(entity.GetExternalSource().GetCidr())
		if err != nil {
			utils.Should(errors.Wrapf(err, "parsing CIDR %s", entity.GetExternalSource().GetCidr()))
			continue
		}

		net := pkgNet.IPNetworkFromIPNet(*ipNet)
		if net.IsValid() {
			if int(net.PrefixLen()) > largestPrefixSoFar {
				largestPrefixSoFar = int(net.PrefixLen())
				ret = entity
			}
		}
	}
	return ret
}

func getLargestSubnets(entities ...*storage.NetworkEntityInfo) []*storage.NetworkEntityInfo {
	tree, err := NewNetworkTreeWrapper(entities)
	utils.Should(errors.Wrap(err, "creating network tree of subnets"))
	// Subnets of Internet node are the matches since rest of the networks are covered by them.
	return tree.GetSubnets(networkgraph.InternetExternalSourceID)
}
