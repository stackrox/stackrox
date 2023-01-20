package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/store"
)

// nodeInventoryHandler is responsible for handling arriving NodeInventory messages, processing them, and sending them to central
type nodeInventoryHandler interface {
	common.SensorComponent
	Stopped() concurrency.ReadOnlyErrorSignal
}

var _ nodeInventoryHandler = (*nodeInventoryHandlerImpl)(nil)

// NewNodeInventoryHandler returns a new instance of a NodeInventoryHandler
func NewNodeInventoryHandler(ch <-chan *storage.NodeInventory, matcher NodeIDMatcher) *nodeInventoryHandlerImpl {
	return &nodeInventoryHandlerImpl{
		inventories: ch,
		toCentral:   nil,
		lock:        &sync.Mutex{},
		stopper:     concurrency.NewStopper(),
		nodeMatcher: matcher,
	}
}

// NodeIDMatcher helps finding NodeWrap by name
type NodeIDMatcher interface {
	GetNodeResource(nodename string) *store.NodeWrap
}

// NodeIDMatcherImpl finds Node by name within NodeStore
type NodeIDMatcherImpl struct {
	nodeStore store.NodeStore
}

// NewNodeIDMatcher creates a NodeIDMatcherImpl
func NewNodeIDMatcher(store store.NodeStore) *NodeIDMatcherImpl {
	return &NodeIDMatcherImpl{
		nodeStore: store,
	}
}

// GetNodeResource returns NodeWrap if a Node with matching name has been found
func (c *NodeIDMatcherImpl) GetNodeResource(nodename string) *store.NodeWrap {
	return c.nodeStore.GetNode(nodename)
}
