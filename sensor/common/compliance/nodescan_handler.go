package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// NodeInventoryHandler is responsible for handling arriving NodeInventory messages, processing them, and sending them to central
type NodeInventoryHandler interface {
	common.SensorComponent
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewNodeInventoryHandler returns a new instance of a NodeInventoryHandler
func NewNodeInventoryHandler(ch <-chan *storage.NodeInventory) NodeInventoryHandler {
	return &nodeScanHandlerImpl{
		inventories: ch,
		toCentral:   nil,
		stopC:       concurrency.NewErrorSignal(),
		lock:        &sync.Mutex{},
		stoppedC:    concurrency.NewErrorSignal(),
	}
}
