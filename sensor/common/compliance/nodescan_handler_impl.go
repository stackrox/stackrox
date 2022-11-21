package compliance

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type nodeScanHandlerImpl struct {
	nodeScans <-chan *storage.NodeScanV2
	toCentral <-chan *central.MsgFromSensor
	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock *sync.Mutex

	stopC concurrency.ErrorSignal
	// runFinishedC is signaled when the goroutine inside of run() finishes
	runFinishedC concurrency.ErrorSignal
	startedC     concurrency.Signal
}

func (c *nodeScanHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.runFinishedC
}

func (c *nodeScanHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NodeScanningCap}
}

func (c *nodeScanHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.toCentral
}

func (c *nodeScanHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.startedC.IsDone() {
		return errors.New("running handler must be stopped before it can be started again")
	}
	c.startedC.Signal()
	if !c.stopC.IsDone() {
		c.toCentral = c.run()
	}
	return nil
}

func (c *nodeScanHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
	c.startedC.Reset()
}

func (c *nodeScanHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeScanHandlerImpl) run() <-chan *central.MsgFromSensor {
	toC := make(chan *central.MsgFromSensor)
	go func() {
		defer c.runFinishedC.Signal()
		defer close(toC)
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
				c.sendScan(toC, scan)
			}
		}
	}()
	return toC
}

func (c *nodeScanHandlerImpl) sendScan(toC chan *central.MsgFromSensor, scan *storage.NodeScanV2) {
	if scan == nil {
		return
	}
	select {
	case <-c.stopC.Done():
	case toC <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_NodeScanV2{
					NodeScanV2: scan,
				},
			},
		},
	}:
	}
}
