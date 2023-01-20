package mocks

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/store"
)

// MockNodeStore is a mock of NodeStore interface.
type MockNodeStore struct {
	mutex sync.RWMutex
	nodes map[string]*store.NodeWrap
}

func NewMockNodeStore() *MockNodeStore {
	return &MockNodeStore{
		mutex: sync.RWMutex{},
		nodes: make(map[string]*store.NodeWrap),
	}
}

// GetNode mocks base method.
func (m *MockNodeStore) GetNode(nodeName string) *store.NodeWrap {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.nodes[nodeName]
}

// AddOrUpdateNode mocks base method.
func (m *MockNodeStore) AddOrUpdateNode(node *store.NodeWrap) bool {
	var oldNode *store.NodeWrap
	concurrency.WithLock(&m.mutex, func() {
		oldNode = m.nodes[node.Name]
		m.nodes[node.Name] = node
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

// RemoveNode mocks base method.
func (m *MockNodeStore) RemoveNode(node *storage.Node) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.nodes, node.Name)
}

// GetNodes mocks base method.
func (m *MockNodeStore) GetNodes() []*store.NodeWrap {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*store.NodeWrap, 0, len(m.nodes))
	for _, node := range m.nodes {
		result = append(result, node)
	}
	return result
}
