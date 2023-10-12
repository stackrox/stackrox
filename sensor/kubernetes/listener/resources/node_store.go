package resources

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/store"
	v1 "k8s.io/api/core/v1"
)

type nodeWrap struct {
	*v1.Node
	addresses []net.IPAddress
}

func wrapNode(node *v1.Node) *nodeWrap {
	wrap := &nodeWrap{Node: node}
	for _, nodeAddr := range node.Status.Addresses {
		if nodeAddr.Type != v1.NodeInternalIP && nodeAddr.Type != v1.NodeExternalIP {
			continue
		}
		parsedIP := net.ParseIP(nodeAddr.Address)
		if parsedIP.IsValid() {
			wrap.addresses = append(wrap.addresses, parsedIP)
		}
	}
	sort.Slice(wrap.addresses, func(i, j int) bool {
		return net.IPAddressLess(wrap.addresses[i], wrap.addresses[j])
	})
	return wrap
}

// nodeStore represents a collection of NodeWraps
type nodeStore interface {
	addOrUpdateNode(node *nodeWrap) bool
	removeNode(node *storage.Node)
	getNode(nodeName string) *nodeWrap
	getNodes() []*nodeWrap
}

var _ nodeStore = (*nodeStoreImpl)(nil)
var _ store.NodeStore = (*nodeStoreImpl)(nil)

// nodeStoreImpl stores nodes in memory
type nodeStoreImpl struct {
	mutex sync.RWMutex
	nodes map[string]*nodeWrap
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (s *nodeStoreImpl) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	if resType != deduper.TypeNode.String() {
		return "", errors.Errorf("Invalid resource type: %v", resType)
	}
	for _, n := range s.getNodes() {
		if string(n.UID) == resID {
			return "", nil
		}
	}
	// Resource on Central but not on Sensor, send for deletion
	return resID, nil
}

func newNodeStore() *nodeStoreImpl {
	return &nodeStoreImpl{
		nodes: make(map[string]*nodeWrap),
	}
}

// Cleanup deletes all entries from store
func (s *nodeStoreImpl) Cleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nodes = make(map[string]*nodeWrap)
}

// addOrUpdateNode upserts node into store.
// It returns true if the IP addresses of the node changed as a result.
func (s *nodeStoreImpl) addOrUpdateNode(node *nodeWrap) bool {
	var oldNode *nodeWrap
	concurrency.WithLock(&s.mutex, func() {
		oldNode = s.nodes[node.Name]
		s.nodes[node.Name] = node
	})

	if oldNode == nil || len(oldNode.addresses) != len(node.addresses) {
		return true
	}
	for i, oldAddr := range oldNode.addresses {
		if oldAddr != node.addresses[i] {
			return true
		}
	}
	return false
}

// removeNode removes node from the store
func (s *nodeStoreImpl) removeNode(node *storage.Node) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.nodes, node.Name)
}

// getNode returns nodeWrap with a given name
func (s *nodeStoreImpl) getNode(nodeName string) *nodeWrap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.nodes[nodeName]
}

// GetNode returns node with a given name or nil if not found
func (s *nodeStoreImpl) GetNode(nodeName string) *storage.Node {
	if wrap := s.getNode(nodeName); wrap != nil {
		return buildNode(wrap.Node)
	}
	return nil
}

// getNodes returns a slice with all nodes stored in the store
func (s *nodeStoreImpl) getNodes() []*nodeWrap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]*nodeWrap, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result
}
