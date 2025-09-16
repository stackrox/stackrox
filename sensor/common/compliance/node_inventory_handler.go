package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
)

var _ common.ComplianceComponent = (*nodeInventoryHandlerImpl)(nil)

// NewNodeInventoryHandler returns a new instance of a NodeInventoryHandler
func NewNodeInventoryHandler(ch <-chan *storage.NodeInventory, iw <-chan *index.IndexReportWrap, nodeIDMatcher NodeIDMatcher, nodeRHCOSmatcher NodeRHCOSMatcher) *nodeInventoryHandlerImpl {
	return &nodeInventoryHandlerImpl{
		inventories:      ch,
		reportWraps:      iw,
		toCentral:        nil,
		centralReady:     concurrency.NewSignal(),
		toCompliance:     nil,
		acksFromCentral:  nil,
		lock:             &sync.Mutex{},
		stopper:          concurrency.NewStopper(),
		nodeMatcher:      nodeIDMatcher,
		nodeRHCOSMatcher: nodeRHCOSmatcher,
		archCache:        make(map[string]string),
	}
}
