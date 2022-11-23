package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
)

// NodeScanHandler is responsible for handling the arriving NodeScanV2 messages, processing then, and sending them to central
type NodeScanHandler interface {
	common.SensorComponent
}

// NewNodeScanHandler returns a new instance of a NodeScanHandler
func NewNodeScanHandler(ch <-chan *storage.NodeScanV2) NodeScanHandler {
	return &nodeScanHandlerImpl{
		nodeScans: ch,
		toCentral: make(chan *central.MsgFromSensor),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
