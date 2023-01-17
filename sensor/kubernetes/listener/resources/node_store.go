package resources

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
	v1 "k8s.io/api/core/v1"
)

func wrapNode(node *v1.Node) *store.NodeWrap {
	wrap := &store.NodeWrap{Node: buildNode(node)}
	for _, nodeAddr := range node.Status.Addresses {
		if nodeAddr.Type != v1.NodeInternalIP && nodeAddr.Type != v1.NodeExternalIP {
			continue
		}
		parsedIP := net.ParseIP(nodeAddr.Address)
		if parsedIP.IsValid() {
			wrap.Addresses = append(wrap.Addresses, parsedIP)
		}
	}
	sort.Slice(wrap.Addresses, func(i, j int) bool {
		return net.IPAddressLess(wrap.Addresses[i], wrap.Addresses[j])
	})
	return wrap
}

// nodeStoreImpl stores nodes in memory
type nodeStoreImpl struct {
	mutex sync.RWMutex
	nodes map[string]*store.NodeWrap
}

// newNodeStore provides a nodeStoreImpl instance
func newNodeStore() *nodeStoreImpl {
	return &nodeStoreImpl{
		nodes: make(map[string]*store.NodeWrap),
	}
}

// AddOrUpdateNode upserts node into store.
// It returns true if the IP addresses of the node changed as a result.
func (s *nodeStoreImpl) AddOrUpdateNode(node *store.NodeWrap) bool {
	var oldNode *store.NodeWrap
	concurrency.WithLock(&s.mutex, func() {
		oldNode = s.nodes[node.Name]
		s.nodes[node.Name] = node
	})

	if oldNode == nil || len(oldNode.Addresses) != len(node.Addresses) {
		return true
	}
	for i, oldAddr := range oldNode.Addresses {
		if oldAddr != node.Addresses[i] {
			return true
		}
	}
	return false
}

// RemoveNode removes node from the store
func (s *nodeStoreImpl) RemoveNode(node *storage.Node) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.nodes, node.Name)
}

// GetNode returns node with a given name
func (s *nodeStoreImpl) GetNode(nodeName string) *store.NodeWrap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.nodes[nodeName]
}

// GetNodes returns a slice with all nodes stored in the store
func (s *nodeStoreImpl) GetNodes() []*store.NodeWrap {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]*store.NodeWrap, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result
}
