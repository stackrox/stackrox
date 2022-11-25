package compliance

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// NodeScanHandler is responsible for handling the arriving NodeScanV2 messages, processing then, and sending them to central
type NodeScanHandler interface {
	common.SensorComponent
	Stopped() concurrency.ReadOnlyErrorSignal
}

// NewNodeScanHandler returns a new instance of a NodeScanHandler
func NewNodeScanHandler(ch <-chan *storage.NodeScanV2) NodeScanHandler {
	return &nodeScanHandlerImpl{
		nodeScans: ch,
		toCentral: nil,
		stopC:     concurrency.NewErrorSignal(),
		lock:      &sync.Mutex{},
		stoppedC:  concurrency.NewErrorSignal(),
	}
}
