package compliance

import (
	"fmt"

	"github.com/stackrox/rox/sensor/common/store"
)

// nodeStore provides functionality to get nodes
type nodeStore interface {
	AddOrUpdateNode(node *store.NodeWrap) bool
	GetNode(nodeName string) *store.NodeWrap
}

// NodeIDMatcher helps finding NodeWrap by name
type NodeIDMatcher interface {
	GetNodeID(nodename string) (string, error)
}

// NodeIDMatcherImpl finds Node by name within NodeStore
type NodeIDMatcherImpl struct {
	nodeStore nodeStore
}

// NewNodeIDMatcher creates a NodeIDMatcherImpl
func NewNodeIDMatcher(store nodeStore) *NodeIDMatcherImpl {
	return &NodeIDMatcherImpl{
		nodeStore: store,
	}
}

// GetNodeID returns NodeID if a Node with matching name has been found
func (c *NodeIDMatcherImpl) GetNodeID(nodename string) (string, error) {
	if node := c.nodeStore.GetNode(nodename); node != nil {
		return node.GetId(), nil
	}
	return "", fmt.Errorf("cannot find node with name '%s'", nodename)
}
