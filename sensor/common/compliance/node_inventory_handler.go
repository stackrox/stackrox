package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
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
