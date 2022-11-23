package compliance

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var errStartMoreThanOnce = errors.New("unable to start the component more than once")

type nodeScanHandlerImpl struct {
	nodeScans <-chan *storage.NodeScanV2
	toCentral <-chan *central.MsgFromSensor

	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock      *sync.Mutex
	numStarts uint32
	stopC     concurrency.ErrorSignal
	// stoppedC is signaled when the goroutine inside of run() finishes
	stoppedC concurrency.ErrorSignal
}

func (c *nodeScanHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
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
	if !atomic.CompareAndSwapUint32(&c.numStarts, 0, 1) {
		return errStartMoreThanOnce
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.toCentral = c.run()
	return nil
}

func (c *nodeScanHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
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
		defer func() {
			c.stoppedC.SignalWithError(c.stopC.Err())
		}()
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
