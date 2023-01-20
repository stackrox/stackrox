package mocks

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/store"
)

// MockNodeIDMatcher always finds a node when GetNodeResource is called
type MockNodeIDMatcher struct {
	nodeStore store.NodeStore
}

// NewMockNodeIDMatcher builds MockNodeIDMatcher
func NewMockNodeIDMatcher(store store.NodeStore) *MockNodeIDMatcher {
	return &MockNodeIDMatcher{
		nodeStore: store,
	}
}

// GetNodeResource always returns a node with give name and hardcoded ID
func (c *MockNodeIDMatcher) GetNodeResource(nodename string) *store.NodeWrap {
	return &store.NodeWrap{Node: &storage.Node{Name: nodename, Id: "abc"}}
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
