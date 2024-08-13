package compliance

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

var _ common.ComplianceComponent = (*nodeInventoryHandlerImpl)(nil)

// NewNodeInventoryHandler returns a new instance of a NodeInventoryHandler
func NewNodeInventoryHandler(ch <-chan *storage.NodeInventory, iw <-chan *IndexReportWrap, matcher NodeIDMatcher) *nodeInventoryHandlerImpl {
	return &nodeInventoryHandlerImpl{
		inventories:     ch,
		reportWraps:     iw,
		toCentral:       nil,
		centralReady:    concurrency.NewSignal(),
		toCompliance:    nil,
		acksFromCentral: nil,
		lock:            &sync.Mutex{},
		stopper:         concurrency.NewStopper(),
		nodeMatcher:     matcher,
	}
}

// IndexReportWrap wraps a v4.IndexReport with additional fields required by Sensor and Central
type IndexReportWrap struct {
	NodeName    string
	NodeID      string
	IndexReport *v4.IndexReport
}
