package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

var _ common.ComplianceComponent = (*nodeInventoryHandlerImpl)(nil)

// NewNodeInventoryHandler returns a new instance of a NodeInventoryHandler
func NewNodeInventoryHandler(ch <-chan *storage.NodeInventory, matcher NodeIDMatcher) *nodeInventoryHandlerImpl {
	return &nodeInventoryHandlerImpl{
		inventories:     ch,
		toCentral:       nil,
		centralReady:    concurrency.NewSignal(),
		toCompliance:    nil,
		acksFromCentral: nil,
		lock:            &sync.Mutex{},
		stopper:         concurrency.NewStopper(),
		nodeMatcher:     matcher,
	}
}
