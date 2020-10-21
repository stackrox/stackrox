package networkgraph

import (
	"bytes"
	"net"
	"sort"

	"github.com/pkg/errors"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "0:0:0:0:0:ffff:0:0/0"
)

// NetworkTree represents a tree of unique networks where every node's children are fully contained networks,
// thereby representing supernet-subnet relationship.
type NetworkTree struct {
	root   *node
	family pkgNet.Family

	lock sync.RWMutex
}

type node struct {
	key      string
	ipNet    *net.IPNet
	children map[string]*node
}

// NewDefaultNetworkTree returns a new instance of NetworkTree for supplied IP address family.
func NewDefaultNetworkTree(family pkgNet.Family) (*NetworkTree, error) {
	root, err := createRoot(family)
	if err != nil {
		return nil, err
	}

	return &NetworkTree{
		root:   root,
		family: family,
	}, nil
}

// NewNetworkTree returns a new instance of NetworkTree built with supplied networks for given IP
func NewNetworkTree(networks map[string]string, family pkgNet.Family) (*NetworkTree, error) {
	t, err := NewDefaultNetworkTree(family)
	if err != nil {
		return nil, err
	}

	if err := t.build(networks); err != nil {
		return nil, err
	}
	return t, nil
}

func createRoot(family pkgNet.Family) (*node, error) {
	var ipNet *net.IPNet
	if family == pkgNet.IPv4 {
		_, ipNet, _ = net.ParseCIDR(ipv4InternetCIDR)
	} else if family == pkgNet.IPv6 {
		_, ipNet, _ = net.ParseCIDR(ipv6InternetCIDR)
	} else {
		return nil, errors.New("failed to create network tree. Invalid IP address family provided")
	}

	return &node{
		key:      InternetExternalSourceID,
		ipNet:    ipNet,
		children: make(map[string]*node),
	}, nil
}

func (t *NetworkTree) build(networks map[string]string) error {
	netSlice := make([]pkgNet.IPNetwork, 0, len(networks))
	netToKey := make(map[pkgNet.IPNetwork]string)
	for key, cidr := range networks {
		ipNet := pkgNet.IPNetworkFromCIDR(cidr)
		if !ipNet.IsValid() {
			return errors.Errorf("received invalid CIDR %s to insert", ipNet.String())
		}
		netToKey[ipNet] = key
		netSlice = append(netSlice, ipNet)
	}

	// Sort the network by prefix length to reduce the tree re-arrangement.
	if t.family == pkgNet.IPv4 {
		sort.Sort(sortableIPv4NetworkSlice(netSlice))
	} else if t.family == pkgNet.IPv6 {
		sort.Sort(sortableIPv6NetworkSlice(netSlice))
	}

	for _, ipNet := range netSlice {
		if err := t.Insert(netToKey[ipNet], ipNet.String()); err != nil {
			return err
		}
	}
	return nil
}

// Insert add a network represent by given key-cidr pair. If the key is already present in the tree, the cidr is updated
// and the tree is rearranged to maintain the supernet-subnet relationship.
func (t *NetworkTree) Insert(key, cidr string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if key == "" {
		return errors.New("received invalid key to insert")
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.New("received invalid CIDR block to insert")
	}
	if _, bits := ipNet.Mask.Size(); bits != t.family.Bits() {
		return errors.Errorf("received invalid CIDR. Expected %s CIDR", t.family.String())
	}

	if oldNode := t.getNodeByKeyNoLock(t.root, key); oldNode != nil {
		// Skip insert if key-value already present, else recreate.
		if oldNode.ipNet.IP.Equal(ipNet.IP) && bytes.Equal(oldNode.ipNet.Mask, ipNet.Mask) {
			return nil
		}
		t.removeNodeNoLock(t.root, key)
	}

	newNode := &node{
		key:      key,
		ipNet:    ipNet,
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
			newNode.key, newNode.ipNet.String(), curr.key)
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

	curr.children[newNode.key] = newNode

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
		if key == curr.key {
			continue
		}

		if netutil.IsIPNetSubset(curr.ipNet, neighbor.ipNet) {
			curr.children[key] = neighbor
			delete(parent.children, key)
		}
	}
}

// Remove removes the network from tree for given key, if present.
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
		return
	}

	for _, node := range curr.children {
		t.removeNodeNoLock(node, key)
	}
}

// GetCIDR returns the cidr (network) for given key, if present.
func (t *NetworkTree) GetCIDR(key string) string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if node := t.getNodeByKeyNoLock(t.root, key); node != nil {
		return node.ipNet.String()
	}
	return ""
}

// GetSubnets returns all the direct subnets (successor) contained by the network for given key, if present.
func (t *NetworkTree) GetSubnets(key string) []string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	curr := t.root
	match := t.getNodeByKeyNoLock(t.root, key)
	if match == nil {
		return nil
	}

	ret := make([]string, 0, len(curr.children))
	for _, child := range match.children {
		ret = append(ret, child.key)
	}
	return ret
}

func (t *NetworkTree) getNodeByKeyNoLock(curr *node, key string) *node {
	if curr == nil {
		return nil
	}

	if curr.key == key {
		return curr
	}

	if node, ok := curr.children[key]; ok {
		return node
	}

	for _, child := range curr.children {
		if node := t.getNodeByKeyNoLock(child, key); node != nil {
			return node
		}
	}
	return nil
}

// GetSupernet returns the direct supernet (predecessor) that fully contains the network for given key, if present.
func (t *NetworkTree) GetSupernet(key string) string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// Supernet of INTERNET is INTERNET.
	if t.root.key == key {
		return key
	}

	match := t.getParentNoLock(t.root, key)
	if match == nil {
		return ""
	}
	return match.key
}

func (t *NetworkTree) getParentNoLock(curr *node, key string) *node {
	if curr == nil {
		return nil
	}

	if _, ok := curr.children[key]; ok {
		return curr
	}

	for _, child := range curr.children {
		if node := t.getParentNoLock(child, key); node != nil {
			return node
		}
	}
	return nil
}

// Search return true if the network for the given key is found in the tree.
func (t *NetworkTree) Search(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getNodeByKeyNoLock(t.root, key) != nil
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
