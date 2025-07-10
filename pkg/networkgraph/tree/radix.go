package tree

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

type nRadixTree struct {
	root   *nRadixNode
	family pkgNet.Family
	// valueNodes points to all the nodes in the tree that hold non-empty values (networks). This facilitates O(1) lookups by key.
	valueNodes map[string]*nRadixNode

	lock sync.RWMutex
}

type nRadixNode struct {
	left, right, parent *nRadixNode
	value               *storage.NetworkEntityInfo
}

// NewDefaultNRadixTree returns a new radix tree for networks.
func NewDefaultNRadixTree(family pkgNet.Family) NetworkTree {
	return newDefaultNRadixTree(family)
}

// NewNRadixTree builds and returns a new radix tree for given networks.
func NewNRadixTree(family pkgNet.Family, entities []*storage.NetworkEntityInfo) (NetworkTree, error) {
	t := newDefaultNRadixTree(family)
	if err := t.build(entities); err != nil {
		return nil, err
	}
	return t, nil
}

func newDefaultNRadixTree(family pkgNet.Family) *nRadixTree {
	if family == pkgNet.InvalidFamily {
		utils.Should(errors.New("failed to create network tree. Invalid IP address family provided"))
	}

	root := &nRadixNode{
		value: networkgraph.InternetProtoWithDesc(family),
	}

	tree := &nRadixTree{
		family:     family,
		root:       root,
		valueNodes: make(map[string]*nRadixNode),
	}
	tree.valueNodes[root.value.GetId()] = tree.root
	return tree
}

func (t *nRadixTree) build(entities []*storage.NetworkEntityInfo) error {
	for _, e := range entities {
		if err := t.insertNoLock(e); err != nil {
			return err
		}
	}
	return nil
}

func (t *nRadixTree) Cardinality() int {
	return len(t.valueNodes)
}

func (t *nRadixTree) GetSupernet(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetNoLock(key, nil)
}

func (t *nRadixTree) GetMatchingSupernet(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetNoLock(key, pred)
}

func (t *nRadixTree) GetSupernetForCIDR(cidr string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetForCIDRNoLock(cidr, nil)
}

func (t *nRadixTree) GetMatchingSupernetForCIDR(cidr string, supernetPred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getMatchingSupernetForCIDRNoLock(cidr, supernetPred)
}

func (t *nRadixTree) GetSubnets(key string) []*storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	node := t.valueNodes[key]
	if node == nil {
		return nil
	}

	subnets := t.getSuccessorWithValsNoLock(node, node)

	results := make([]*storage.NetworkEntityInfo, 0, len(subnets))
	for _, n := range subnets {
		results = append(results, n.CloneVT())
	}
	return results
}

func (t *nRadixTree) GetSubnetsForCIDR(cidr string) []*storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Errorf("Could not parse CIDR. CIDR %s is invalid: %v", cidr, err.Error())
		return nil
	}

	startNode := t.findStartNodeNoLock(ipNet)
	subnets := t.getSuccessorWithValsNoLock(startNode, startNode)

	results := make([]*storage.NetworkEntityInfo, 0, len(subnets))
	for _, n := range subnets {
		results = append(results, n.CloneVT())
	}
	return results
}

func (t *nRadixTree) findStartNodeNoLock(ipNet *net.IPNet) *nRadixNode {
	bit := byte(0x80)
	node := t.root
	i := 0
	for node != nil {
		if ipNet.Mask[i]&bit == 0 {
			break
		}

		if ipNet.IP[i]&bit != 0 {
			node = node.right
		} else {
			node = node.left
		}

		if bit >>= 1; bit == 0 {
			if i++; i >= len(ipNet.IP) {
				break
			}
			bit = byte(0x80)
		}
	}
	return node
}

func (t *nRadixTree) getSuccessorWithValsNoLock(startNode *nRadixNode, curr *nRadixNode) []*storage.NetworkEntityInfo {
	if startNode == nil || curr == nil {
		return nil
	}

	if startNode != curr && curr.value != nil {
		return []*storage.NetworkEntityInfo{curr.value}
	}

	var ret []*storage.NetworkEntityInfo
	ret = append(ret, t.getSuccessorWithValsNoLock(startNode, curr.left)...)
	ret = append(ret, t.getSuccessorWithValsNoLock(startNode, curr.right)...)
	return ret
}

func (t *nRadixTree) Get(key string) *storage.NetworkEntityInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	node := t.valueNodes[key]
	if node == nil {
		return nil
	}

	ret := node.value.CloneVT()
	// Internet entity is expected only with ID and Type fields.
	rmDescIfInternet(ret)
	return ret
}

func (t *nRadixTree) getMatchingSupernetNoLock(key string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	if t.root.value.GetId() == key {
		ret := t.root.value.CloneVT()
		rmDescIfInternet(ret)
		return ret
	}

	node := t.valueNodes[key]
	if node == nil {
		return nil
	}

	match := t.getMatchingParentNoLock(node, pred)
	if match == nil {
		return nil
	}

	ret := match.value.CloneVT()
	rmDescIfInternet(ret)
	return ret
}

func (t *nRadixTree) getMatchingSupernetForCIDRNoLock(cidr string, pred func(entity *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Errorf("Could not parse CIDR. CIDR %s is invalid: %v", cidr, err.Error())
		return nil
	}

	match, err := t.findCIDRNoLock(ipNet)
	if err != nil {
		return nil
	}

	_, matchedIPNet, err := net.ParseCIDR(match.value.GetExternalSource().GetCidr())
	if err != nil {
		return nil
	}

	// Matched network could be exact match or supernet fully containing the CIDR block. Latter case is the result.
	if !ipNetEqual(matchedIPNet, ipNet) {
		if pred == nil || pred(match.value) {
			return match.value
		}
	}

	// If the matched network is an incoming CIDR block, continue looking for its parent that satisfies the predicate.
	match = t.getMatchingParentNoLock(match, pred)
	if match == nil {
		return nil
	}

	ret := match.value.CloneVT()
	rmDescIfInternet(ret)
	return ret
}

func (t *nRadixTree) getMatchingParentNoLock(curr *nRadixNode, pred func(entity *storage.NetworkEntityInfo) bool) *nRadixNode {
	if curr == nil || curr.parent == nil {
		return nil
	}
	if curr.parent.value != nil {
		if pred == nil || pred(curr.parent.value) {
			return curr.parent
		}
	}
	return t.getMatchingParentNoLock(curr.parent, pred)
}

func (t *nRadixTree) Exists(key string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.valueNodes[key] != nil
}

func (t *nRadixTree) Insert(entity *storage.NetworkEntityInfo) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.insertNoLock(entity)
}

// Inserts a network into radix tree. If the network already exists, insertion fails.
func (t *nRadixTree) insertNoValidate(ipNet *net.IPNet, value *storage.NetworkEntityInfo) error {
	bit := byte(0x80)
	node := t.root
	next := t.root
	i := 0
	// Traverse the tree for the bits that already exist in the tree.
	for bit&ipNet.Mask[i] != 0 {
		// If the bit is set, go right, otherwise left.
		if ipNet.IP[i]&bit != 0 {
			next = node.right
		} else {
			next = node.left
		}

		if next == nil {
			break
		}
		node = next

		if bit >>= 1; bit == 0 {
			// All the bits (32/128) have been walked, stop.
			if i++; i >= len(ipNet.IP) {
				break
			}
			// Reset and move to lower part.
			bit = byte(0x80)
		}
	}

	// If finished walking network bits of mask and a node already exist, try updating it with the value.
	if next != nil {
		// Node already filled. Indicate that the new node was not actually inserted.
		if node.value != nil {
			return errors.Errorf("CIDR %s conflicts with existing CIDR %s in the network tree",
				value.GetExternalSource().GetCidr(), node.value.GetExternalSource().GetCidr())
		}
		node.value = value
		t.valueNodes[value.GetId()] = node
		return nil
	}

	// There still are bits to be walked, so go ahead and add them to the tree.
	for bit&ipNet.Mask[i] != 0 {
		next = &nRadixNode{}
		next.parent = node
		if ipNet.IP[i]&bit != 0 {
			node.right = next
		} else {
			node.left = next
		}

		node = next

		if bit >>= 1; bit == 0 {
			if i++; i >= len(ipNet.IP) {
				break
			}
			bit = byte(0x80)
		}
	}
	node.value = value
	t.valueNodes[value.GetId()] = node
	return nil
}

func (t *nRadixTree) insertNoLock(entity *storage.NetworkEntityInfo) error {
	ipNet, err := t.validateEntity(entity)
	if err != nil {
		return errors.Wrapf(err, "failed to insert entity %s", entity.GetId())
	}
	return t.insertNoValidate(ipNet, entity)
}

func (t *nRadixTree) validateEntity(e *storage.NetworkEntityInfo) (*net.IPNet, error) {
	if e.GetId() == "" {
		return nil, errors.New("received entity without ID")
	}

	if !networkgraph.IsExternal(e) {
		return nil, errors.New("received entity with incorrect type; must be INTERNET or EXTERNAL_SOURCE")
	}

	_, ipNet, err := net.ParseCIDR(e.GetExternalSource().GetCidr())
	if err != nil {
		return nil, errors.Wrap(err, "received invalid CIDR block")
	}

	if _, bits := ipNet.Mask.Size(); bits != t.family.Bits() {
		return nil, errors.Errorf("received invalid CIDR. Expected %s CIDR", t.family.String())
	}
	return ipNet, nil
}

// Returns the smallest subnet larger than or equal to the queried address.
func (t *nRadixTree) findCIDRNoLock(ipNet *net.IPNet) (*nRadixNode, error) {
	var ret *nRadixNode
	bit := byte(0x80)
	node := t.root
	i := 0
	for node != nil {
		if node.value != nil {
			ret = node
		}

		if ipNet.IP[i]&bit != 0 {
			node = node.right
		} else {
			node = node.left
		}

		if node == nil {
			break
		}

		// All network bits are traversed. If a supernet was found along the way, `ret` holds it,
		// else there does not exist any supernet containing the search network/address.
		if ipNet.Mask[i]&bit == 0 {
			break
		}

		if bit >>= 1; bit == 0 {
			if i++; i >= len(ipNet.IP) {
				if node.value != nil {
					ret = node
				}
				break
			}
			bit = byte(0x80)
		}
	}

	return ret, nil
}

// Checks that all leaf nodes have values. All leaves
// should have nodes since paths represent values.
func validateLeavesHaveValues(node *nRadixNode) bool {
	if node == nil {
		return true
	}

	if node.left == nil && node.right == nil && node.value == nil {
		return false
	}

	if node.left != nil && !validateLeavesHaveValues(node.left) {
		return false
	}

	if node.right != nil && !validateLeavesHaveValues(node.right) {
		return false
	}

	return true
}

func getCardinalityByValues(node *nRadixNode) int {
	if node == nil {
		return 0
	}

	cardinality := 0

	if node.value != nil {
		cardinality = 1
	}

	if node.left != nil {
		cardinality += getCardinalityByValues(node.left)
	}

	if node.right != nil {
		cardinality += getCardinalityByValues(node.right)
	}

	return cardinality
}

// Checks that the number of values in the network tree
// matches the number of keys in valueNodes.
func (t *nRadixTree) validateCardinality() bool {
	return getCardinalityByValues(t.root) == t.Cardinality()
}

func cloneIPNet(ipNet *net.IPNet) *net.IPNet {
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)

	mask := make(net.IPMask, len(ipNet.Mask))
	copy(mask, ipNet.Mask)

	return &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
}

func equalValueIpNet(value *storage.NetworkEntityInfo, ipNet *net.IPNet) bool {
	valueCidr := value.GetExternalSource().GetCidr()
	ipNetCidr := ipNet.String()
	return valueCidr == ipNetCidr
}

func validateValuesRecursive(ipNet *net.IPNet, octetIdx int, bit byte, node *nRadixNode) bool {
	if node.value != nil {
		if !equalValueIpNet(node.value, ipNet) {
			return false
		}
	}

	if octetIdx >= len(ipNet.IP) && node.value == nil {
		return false
	}

	newBit := bit >> 1
	newOctetIdx := octetIdx
	if newBit == 0 {
		newOctetIdx = octetIdx + 1
		newBit = byte(0x80)
	}

	if node.right != nil {
		newIpNet := cloneIPNet(ipNet)
		newIpNet.IP[octetIdx] |= bit
		newIpNet.Mask[octetIdx] |= bit
		valid := validateValuesRecursive(newIpNet, newOctetIdx, newBit, node.right)
		if !valid {
			return false
		}
	}

	if node.left != nil {
		newIpNet := cloneIPNet(ipNet)
		newIpNet.Mask[octetIdx] |= bit
		valid := validateValuesRecursive(newIpNet, newOctetIdx, newBit, node.left)
		if !valid {
			return false
		}
	}

	return true
}

// Checks that the values of nodes correspond to the paths taken from
// the root to the nodes.
func (t *nRadixTree) validateValues() bool {
	ip := make(net.IP, 4)
	mask := make(net.IPMask, 4)

	ipNet := &net.IPNet{
		IP:   ip,
		Mask: mask,
	}

	octetIdx := 0
	bit := byte(0x80)
	return validateValuesRecursive(ipNet, octetIdx, bit, t.root)
}

func (t *nRadixTree) ValidateNetworkTree() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.root == nil {
		return false
	}

	if !validateLeavesHaveValues(t.root) {
		log.Errorf("Found a leaf without a value")
		return false
	}

	if !t.validateCardinality() {
		log.Errorf("The number of values in the tree doesn't match the number of keys")
		return false
	}

	if !t.validateValues() {
		log.Errorf("Values do not match tree path")
		return false
	}

	return true
}

func removeRecursively(node *nRadixNode) {
	// Do not remove the root.
	if node.parent == nil {
		return
	}

	parent := node.parent

	if node.left == nil && node.right == nil && node.value == nil {
		if parent.right == node {
			parent.right = nil
		} else {
			parent.left = nil
		}
		removeRecursively(parent)
	}
}

func (t *nRadixTree) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	node := t.valueNodes[key]
	if node == nil {
		log.Errorf("Network to delete (id=%s) not found in tree. Noop.", key)
		return
	}

	// Do not remove the root.
	if node.parent == nil {
		return
	}

	node.value = nil
	removeRecursively(node)

	delete(t.valueNodes, key)
}
