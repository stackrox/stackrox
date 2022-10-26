package compliance

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

type nodeScanHandlerImpl struct {
	nodeScans <-chan *storage.NodeScanV2
	toCentral chan *central.MsgFromSensor

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (c *nodeScanHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NodeScanningV2}
}

func (c *nodeScanHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return c.toCentral
}

func (c *nodeScanHandlerImpl) Start() error {
	go c.run()
	return nil
}

func (c *nodeScanHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
}

func (c *nodeScanHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
}

func (c *nodeScanHandlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

func (c *nodeScanHandlerImpl) run() {
	defer c.stoppedC.Signal()
	for {
		select {
		case <-c.stopC.Done():
			c.stoppedC.SignalWithError(c.stopC.Err())

		case scan, ok := <-c.nodeScans:
			if !ok {
				c.stoppedC.SignalWithError(errors.New("channel receiving node scans v2 is closed"))
				return
			}
			// Do something with the scan if needed, e.g. attach NodeID
			c.sendScan(scan)
		}
	}
}

func (c *nodeScanHandlerImpl) sendScan(scan *storage.NodeScanV2) {
	select {
	case <-c.stoppedC.Done():
		log.Errorf("failed to send update: %s", proto.MarshalTextString(scan))
		return
	case c.toCentral <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_NodeScanV2{
					NodeScanV2: scan,
				},
			},
		},
	}:
		return
	}
}
