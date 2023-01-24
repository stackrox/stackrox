package mocks

import (
	"github.com/stackrox/rox/sensor/common/store"
)

// nodeStore provides functionality to get nodes
type nodeStore interface {
	AddOrUpdateNode(node *store.NodeWrap) bool
	GetNode(nodeName string) *store.NodeWrap
}

// MockNodeIDMatcher always finds a node when GetNodeResource is called
type MockNodeIDMatcher struct {
	nodeStore nodeStore
}

// NewMockNodeIDMatcher builds MockNodeIDMatcher
func NewMockNodeIDMatcher(store nodeStore) *MockNodeIDMatcher {
	return &MockNodeIDMatcher{
		nodeStore: store,
	}
}

// GetNodeID always returns a node with give name and hardcoded ID
func (c *MockNodeIDMatcher) GetNodeID(nodename string) (string, error) {
	return "abc", nil
}

// MockNodeStore is a thread-unsafe, map-based implementation of in-memory store
type MockNodeStore struct {
	nodes map[string]*store.NodeWrap
}

// NewMockNodeStore returns MockNodeStore
func NewMockNodeStore() *MockNodeStore {
	return &MockNodeStore{
		nodes: make(map[string]*store.NodeWrap),
	}
}

// GetNode retrieves a node
func (m *MockNodeStore) GetNode(nodeName string) *store.NodeWrap {
	return m.nodes[nodeName]
}

// AddOrUpdateNode upserts a node
func (m *MockNodeStore) AddOrUpdateNode(node *store.NodeWrap) bool {
	m.nodes[node.Name] = node
	return false
}
