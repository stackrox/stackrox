package tree

import (
	"bytes"
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "::ffff:0:0/0"
)

// NetworkTree represents a tree of unique networks where every node's children are fully contained networks,
// thereby representing supernet-subnet relationship.
type NetworkTree struct {
	root   *node
	family pkgNet.Family
	// nodes points to all the nodes in the tree to faciliate O(1) access time.
	nodes map[string]*node

	lock sync.RWMutex
}

type node struct {
	ipNet    *net.IPNet
	entity   *storage.NetworkEntityInfo
	children map[string]*node
}

// NewDefaultNetworkTree returns a new instance of NetworkTree for the supplied IP address family.
func NewDefaultNetworkTree(family pkgNet.Family) (*NetworkTree, error) {
	tree := &NetworkTree{
		family: family,
		nodes:  make(map[string]*node),
	}

	if err := tree.addRoot(family); err != nil {
		return nil, err
	}

	return tree, nil
}

// NewNetworkTree returns a new instance of NetworkTree built with supplied networks for given IP address family.
func NewNetworkTree(networks []*storage.NetworkEntityInfo, family pkgNet.Family) (*NetworkTree, error) {
	t, err := NewDefaultNetworkTree(family)
	if err != nil {
		return nil, err
	}

	if err := t.build(networks); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *NetworkTree) addRoot(family pkgNet.Family) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	var ipNet *net.IPNet
	if family == pkgNet.IPv4 {
		_, ipNet, _ = net.ParseCIDR(ipv4InternetCIDR)
	} else if family == pkgNet.IPv6 {
		_, ipNet, _ = net.ParseCIDR(ipv6InternetCIDR)
	} else {
		return errors.New("failed to create network tree. Invalid IP address family provided")
	}

	// Root node is not marked as default as it not known external network, instead represents everything unknown.
	t.root = &node{
		ipNet:    ipNet,
		entity:   networkgraph.InternetEntity().ToProto(),
		children: make(map[string]*node),
	}
	t.addToTopLevelNoLock(t.root)
	return nil
}

func (t *NetworkTree) addToTopLevelNoLock(node *node) {
	t.nodes[node.entity.GetId()] = node
}

func (t *NetworkTree) removeFromTopLevelNoLock(key string) {
	delete(t.nodes, key)
}

func (t *NetworkTree) build(entities []*storage.NetworkEntityInfo) error {
	netSlice := make([]pkgNet.IPNetwork, 0, len(entities))
	ipNetToEntity := make(map[pkgNet.IPNetwork]*storage.NetworkEntityInfo)
	for _, entity := range entities {
		ipNet, err := t.validateEntity(entity)
		if err != nil {
			return errors.Wrap(err, "failed to build network tree")
		}
		ipNetwork := pkgNet.IPNetworkFromIPNet(*ipNet)
		ipNetToEntity[ipNetwork] = entity
		netSlice = append(netSlice, ipNetwork)
	}

	// Sort the network by prefix length to reduce the tree re-arrangement.
	normalizeNetworks(t.family, netSlice)

	for _, ipNet := range netSlice {
		if err := t.Insert(ipNetToEntity[ipNet]); err != nil {
			return err
		}
	}
	return nil
}

func normalizeNetworks(family pkgNet.Family, nets []pkgNet.IPNetwork) {
	if family == pkgNet.IPv4 {
		sort.Sort(sortableIPv4NetworkSlice(nets))
	} else if family == pkgNet.IPv6 {
		sort.Sort(sortableIPv6NetworkSlice(nets))
	}
}

// Insert add the supplied network entity. If a entity with the same key is already present in the tree,
// the CIDR of stored entity is updated and the tree is rearranged to maintain the supernet-subnet relationship.
func (t *NetworkTree) Insert(entity *storage.NetworkEntityInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	ipNet, err := t.validateEntity(entity)
	if err != nil {
		return errors.Wrapf(err, "failed to insert entity %s", entity.GetId())
	}
	return t.insertNoValidate(entity, ipNet)
}

func (t *NetworkTree) validateEntity(entity *storage.NetworkEntityInfo) (*net.IPNet, error) {
	if entity.GetId() == "" {
		return nil, errors.New("received entity without ID")
	}

	if !networkgraph.IsExternal(entity) {
		return nil, errors.New("received entity with incorrect type; must be INTERNET or EXTERNAL_SOURCE")
	}

	_, ipNet, err := net.ParseCIDR(entity.GetExternalSource().GetCidr())
	if err != nil {
		return nil, errors.Wrap(err, "received invalid CIDR block")
	}

	if _, bits := ipNet.Mask.Size(); bits != t.family.Bits() {
		return nil, errors.Errorf("received invalid CIDR. Expected %s CIDR", t.family.String())
	}
	return ipNet, nil
}

func (t *NetworkTree) insertNoValidate(entity *storage.NetworkEntityInfo, ipNet *net.IPNet) error {
	if oldNode := t.nodes[entity.GetId()]; oldNode != nil {
		// Skip insert if key-value already present.
		if oldNode.ipNet.IP.Equal(ipNet.IP) && bytes.Equal(oldNode.ipNet.Mask, ipNet.Mask) {
			return nil
		}
		// If subnet is different, recreate the node as it could be placed at different position in tree.
		t.removeNodeNoLock(t.root, entity.GetId())
	}

	newNode := &node{
		ipNet:    &net.IPNet{IP: ipNet.IP, Mask: ipNet.Mask},
		entity:   entity.Clone(),
		children: make(map[string]*node),
	}

	if _, err := t.insertNodeNoLock(t.root, newNode); err != nil {
		return err
	}
	return nil
}

func (t *NetworkTree) insertNodeNoLock(curr, newNode *node) (bool, error) {
	// INTERNET (root) would always contain any network if no other network contains it.
	if !netutil.IsIPNetSubset(curr.ipNet, newNode.ipNet) {
		return false, nil
	}

	if curr.ipNet.IP.Equal(newNode.ipNet.IP) && bytes.Equal(curr.ipNet.Mask, newNode.ipNet.Mask) {
		return false, errors.Errorf("network %s (CIDR=%s) conflicting with existing network %s in the tree",
			newNode.entity.GetId(), newNode.ipNet.String(), curr.entity.GetId())
	}

	for _, child := range curr.children {
		ok, err := t.insertNodeNoLock(child, newNode)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	curr.children[newNode.entity.GetId()] = newNode
	t.addToTopLevelNoLock(newNode)

	// Arrange neighboring smaller networks as subnet of new network.
	t.neighborsToChildrenNoLock(newNode, curr)
	return true, nil
}

func (t *NetworkTree) neighborsToChildrenNoLock(curr, parent *node) {
	if curr == nil {
		return
	}

	if curr.children == nil {
		curr.children = make(map[string]*node)
	}

	for key, neighbor := range parent.children {
		if key == curr.entity.GetId() {
			continue
		}

		if netutil.IsIPNetSubset(curr.ipNet, neighbor.ipNet) {
			curr.children[key] = neighbor
			delete(parent.children, key)
		}
	}
}

// Remove removes the network entity with given key from tree, if present.
func (t *NetworkTree) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.removeNodeNoLock(t.root, key)
}

func (t *NetworkTree) removeNodeNoLock(curr *node, key string) {
	if curr == nil {
		return
	}

	if child, ok := curr.children[key]; ok {
		for grandChildKey, grandChild := range child.children {
			curr.children[grandChildKey] = grandChild
		}

		delete(curr.children, key)
		t.removeFromTopLevelNoLock(key)
		return
	}

	for _, node := range curr.children {
		t.removeNodeNoLock(node, key)
	}
}

// Get returns the network entity for given key, if present, otherwise nil.
func (t *NetworkTree) Get(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if node := t.nodes[key]; node != nil {
		return node.entity.Clone()
	}
	return nil
}

// GetSubnets returns the largest disjoint subnets contained by the network for given key, if present.
func (t *NetworkTree) GetSubnets(key string) []*storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	match := t.nodes[key]
	if match == nil {
		return nil
	}

	ret := make([]*storage.NetworkEntityInfo, 0, len(match.children))
	for _, child := range match.children {
		ret = append(ret, child.entity.Clone())
	}
	return ret
}

// GetSubnetsForCIDR returns the largest disjoint subnets contained by the given network, if any.
func (t *NetworkTree) GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getSubnetForIPNetNoLock(t.root, ipNet)
}

func (t *NetworkTree) getSubnetForIPNetNoLock(curr *node, queryIPNet *net.IPNet) []*storage.NetworkEntityInfo {
	if curr == nil {
		return nil
	}

	// We are looking for largest subnets that is fully contained by query network.
	if netutil.IsIPNetSubset(queryIPNet, curr.ipNet) {
		return []*storage.NetworkEntityInfo{curr.entity.Clone()}
	}

	var ret []*storage.NetworkEntityInfo
	for _, child := range curr.children {
		ret = append(ret, t.getSubnetForIPNetNoLock(child, queryIPNet)...)
	}
	return ret
}

// GetSupernet returns the smallest supernet that fully contains the network for given key, if present.
func (t *NetworkTree) GetSupernet(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetNoLock(key, nil)
}

// GetMatchingSupernet returns the smallest supernet that fully contains the network for given key and satisfies the predicate.
func (t *NetworkTree) GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetNoLock(key, pred)
}

func (t *NetworkTree) getMatchingSupernetNoLock(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	// Supernet of INTERNET is INTERNET.
	if t.root.entity.GetId() == key {
		return t.root.entity.Clone()
	}

	if node := t.nodes[key]; node == nil {
		return nil
	}

	supernet, _ := t.getMatchingParentNoLock(t.root, key, pred)
	if supernet == nil {
		return nil
	}
	return supernet.entity.Clone()
}

func (t *NetworkTree) getMatchingParentNoLock(curr *node, key string, pred func(entity *storage.NetworkEntityInfo) bool) (*node, *node) {
	if curr == nil {
		return nil, nil
	}

	if child, ok := curr.children[key]; ok {
		if pred == nil || pred(curr.entity) {
			return curr, child
		}
		return nil, child
	}

	for _, node := range curr.children {
		if parent, match := t.getMatchingParentNoLock(node, key, pred); match != nil {
			if parent != nil {
				return parent, match
			}

			if pred(curr.entity) {
				return curr, match
			}
		}
	}
	return nil, nil
}

// GetSupernetForCIDR returns the smallest supernet that fully contains the given network.
func (t *NetworkTree) GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetForIPNetNoLock(t.root, ipNet, nil)
}

// GetMatchingSupernetForCIDR returns the smallest supernet that fully contains the given network and satisfies the predicate.
func (t *NetworkTree) GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}

	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetForIPNetNoLock(t.root, ipNet, supernetPred)
}

func (t *NetworkTree) getMatchingSupernetForIPNetNoLock(curr *node, queryIPNet *net.IPNet, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	if curr == nil {
		return nil
	}

	if !netutil.IsIPNetSubset(curr.ipNet, queryIPNet) {
		return nil
	}

	var supernetSoFar *storage.NetworkEntityInfo
	if supernetPred == nil || supernetPred(curr.entity) {
		supernetSoFar = curr.entity.Clone()
	}

	for _, child := range curr.children {
		if supernet := t.getMatchingSupernetForIPNetNoLock(child, queryIPNet, supernetPred); supernet != nil {
			supernetSoFar = supernet
		}
	}
	return supernetSoFar
}

// Search return true if the network entity for the given key is found in the tree.
func (t *NetworkTree) Search(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, ok := t.nodes[key]
	return ok
}

type sortableIPv4NetworkSlice []pkgNet.IPNetwork

func (s sortableIPv4NetworkSlice) Len() int {
	return len(s)
}

func (s sortableIPv4NetworkSlice) Less(i, j int) bool {
	if s[i].PrefixLen() != s[j].PrefixLen() {
		return s[i].PrefixLen() < s[j].PrefixLen()
	}
	if !s[i].IP().AsNetIP().Equal(s[j].IP().AsNetIP()) {
		return bytes.Compare(s[i].IP().AsNetIP(), s[j].IP().AsNetIP()) > 0
	}
	return false
}

func (s sortableIPv4NetworkSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type sortableIPv6NetworkSlice []pkgNet.IPNetwork

func (s sortableIPv6NetworkSlice) Len() int {
	return len(s)
}

func (s sortableIPv6NetworkSlice) Less(i, j int) bool {
	if s[i].PrefixLen() != s[j].PrefixLen() {
		return s[i].PrefixLen() < s[j].PrefixLen()
	}
	if !s[i].IP().AsNetIP().Equal(s[j].IP().AsNetIP()) {
		return bytes.Compare(s[i].IP().AsNetIP(), s[j].IP().AsNetIP()) > 0
	}
	return false
}

func (s sortableIPv6NetworkSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
