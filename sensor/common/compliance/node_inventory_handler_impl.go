package compliance

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
)

var (
	errInputChanClosed   = errors.New("channel receiving node inventories is closed")
	errStartMoreThanOnce = errors.New("unable to start the component more than once")
)

type nodeInventoryHandlerImpl struct {
	inventories  <-chan *storage.NodeInventory
	toCentral    <-chan *central.MsgFromSensor
	centralReady concurrency.Signal
	toCompliance <-chan *common.MessageToComplianceWithAddress
	nodeMatcher  NodeIDMatcher
	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock    *sync.Mutex
	stopper concurrency.Stopper
}

func (c *nodeInventoryHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

func (c *nodeInventoryHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NodeScanningCap}
}

// ResponsesC returns a channel with messages to Central. It must be called after Start() for the channel to be not nil
func (c *nodeInventoryHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

// ComplianceC returns a channel with messages to Compliance
func (c *nodeInventoryHandlerImpl) ComplianceC() <-chan *common.MessageToComplianceWithAddress {
	return c.toCompliance
}

func (c *nodeInventoryHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil || c.toCompliance != nil {
		return errStartMoreThanOnce
	}
	c.toCentral, c.toCompliance = c.run()
	return nil
}

func (c *nodeInventoryHandlerImpl) Stop(_ error) {
	c.stopper.Client().Stop()
}

func (c *nodeInventoryHandlerImpl) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		c.centralReady.Signal()
	}
}

func (c *nodeInventoryHandlerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent from Central to Sensor (yet).
	// It uses the sensor component so that the lifecycle (start, stop) can be handled when Sensor starts up.
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeInventoryHandlerImpl) run() (<-chan *central.MsgFromSensor, <-chan *common.MessageToComplianceWithAddress) {
	toCentral := make(chan *central.MsgFromSensor)
	toCompliance := make(chan *common.MessageToComplianceWithAddress)
	go func() {
		defer c.stopper.Flow().ReportStopped()
		defer close(toCentral)
		defer close(toCompliance)
		for {
			select {
			case <-c.stopper.Flow().StopRequested():
				return
			case inventory, ok := <-c.inventories:
				if !ok {
					c.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				if !c.centralReady.IsDone() {
					log.Warnf("Received NodeInventory but Central is not reachable. Requesting Compliance to resend NodeInventory later")
					c.sendNackToCompliance(toCompliance, inventory)
					continue
				}
				if inventory == nil {
					log.Warnf("Received nil node inventory: not sending to Central")
					break
				}
				if nodeID, err := c.nodeMatcher.GetNodeID(inventory.GetNodeName()); err != nil {
					log.Infof("Sending NACK to compliance after receiving unknown NodeInventory with ID %s", inventory.GetNodeId())
					c.sendNackToCompliance(toCompliance, inventory)
				} else {
					inventory.NodeId = nodeID
					metrics.ObserveReceivedNodeInventory(inventory)
					log.Infof("Mapping NodeInventory name '%s' to Node ID '%s'", inventory.GetNodeName(), nodeID)
					c.sendNodeInventory(toCentral, inventory)
					log.Infof("Sending ACK to compliance for NodeInventory with ID %s", inventory.GetNodeId())
					c.sendAckToCompliance(toCompliance, inventory)
				}
			}
		}
	}()
	return toCentral, toCompliance
}

func (c *nodeInventoryHandlerImpl) sendNackToCompliance(complianceC chan<- *common.MessageToComplianceWithAddress, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	complianceC <- &common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_Nack{
				Nack: &sensor.MsgToCompliance_NodeInventoryNack{
					NodeId: inventory.GetNodeId(),
				},
			},
		},
		Hostname:  inventory.GetNodeName(),
		Broadcast: false,
	}
}

func (c *nodeInventoryHandlerImpl) sendAckToCompliance(complianceC chan<- *common.MessageToComplianceWithAddress, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	complianceC <- &common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_Ack{
				Ack: &sensor.MsgToCompliance_NodeInventoryAck{
					NodeId: inventory.GetNodeId(),
				},
			},
		},
		Hostname:  inventory.GetNodeName(),
		Broadcast: false,
	}
}

func (c *nodeInventoryHandlerImpl) sendNodeInventory(toC chan<- *central.MsgFromSensor, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	select {
	case <-c.stopper.Flow().StopRequested():
	case toC <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     inventory.GetNodeId(),
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE, // There is no action required for NodeInventory as this is not a K8s resource
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: inventory,
				},
			},
		},
	}:
	}
}
