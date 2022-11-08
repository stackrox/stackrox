package compliance

import (
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
	return []centralsensor.SensorCapability{centralsensor.NodeScanningCap}
}

func (c *nodeScanHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return c.toCentral
}

func (c *nodeScanHandlerImpl) Start() error {
	// c.run closes chan toCentral on exit, so we do not want to close twice in case of a restart
	if !c.stopC.IsDone() {
		go c.run()
		return nil
	}
	return errors.New("stopped handlers cannot be restarted")
}

func (c *nodeScanHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
}

func (c *nodeScanHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
}

func (c *nodeScanHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

func (c *nodeScanHandlerImpl) run() {
	defer c.stoppedC.SignalWithError(c.stopC.Err())
	defer close(c.toCentral)

	for !c.stopC.IsDone() {
		select {
		case <-c.stopC.Done():
			return
		case scan, ok := <-c.nodeScans:
			if !ok {
				c.stopC.SignalWithError(errors.New("channel receiving node scans v2 is closed"))
				return
			}
			// TODO(ROX-12943): Do something with the scan, e.g., attach NodeID
			c.sendScan(scan)
		}
	}
}

func (c *nodeScanHandlerImpl) sendScan(scan *storage.NodeScanV2) {
	if scan != nil {
		c.toCentral <- &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_NodeScanV2{
						NodeScanV2: scan,
					},
				},
			},
		}
	}
}
