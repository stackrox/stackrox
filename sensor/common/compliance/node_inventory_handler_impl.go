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
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	errInputChanClosed   = errors.New("channel receiving node inventories is closed")
	errStartMoreThanOnce = errors.New("unable to start the component more than once")
)

type nodeInventoryHandlerImpl struct {
	inventories  <-chan *storage.NodeInventory
	toCentral    <-chan *message.ExpiringMessage
	centralReady concurrency.Signal
	// acksFromCentral is for connecting the replies from Central with the toCompliance chan
	acksFromCentral chan common.MessageToComplianceWithAddress
	toCompliance    <-chan common.MessageToComplianceWithAddress
	nodeMatcher     NodeIDMatcher
	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock    *sync.Mutex
	stopper concurrency.Stopper
}

func (c *nodeInventoryHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

func (c *nodeInventoryHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ResponsesC returns a channel with messages to Central. It must be called after Start() for the channel to be not nil
func (c *nodeInventoryHandlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return c.toCentral
}

// ComplianceC returns a channel with messages to Compliance
func (c *nodeInventoryHandlerImpl) ComplianceC() <-chan common.MessageToComplianceWithAddress {
	return c.toCompliance
}

func (c *nodeInventoryHandlerImpl) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.toCentral != nil || c.toCompliance != nil {
		return errStartMoreThanOnce
	}
	c.acksFromCentral = make(chan common.MessageToComplianceWithAddress)
	c.toCentral, c.toCompliance = c.run()
	return nil
}

func (c *nodeInventoryHandlerImpl) Stop(_ error) {
	if !c.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = c.stopper.Client().Stopped().Wait()
			close(c.acksFromCentral)
		}()
	}
	c.stopper.Client().Stop()
}

func (c *nodeInventoryHandlerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		c.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		// As Compliance enters a retry loop when it is not receiving an ACK,
		// there is no need to do anything when entering offline mode
		c.centralReady.Reset()
	}
}

func (c *nodeInventoryHandlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	ackMsg := msg.GetNodeInventoryAck()
	if ackMsg == nil {
		return nil
	}
	log.Debugf("Received node-scanning-ACK message: %v", ackMsg)
	metrics.ObserveNodeInventoryAck(ackMsg.GetNodeName(), ackMsg.GetAction().String(),
		metrics.AckReasonUnknown, metrics.AckOriginCentral)
	switch ackMsg.GetAction() {
	case central.NodeInventoryACK_ACK:
		c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(), sensor.MsgToCompliance_NodeInventoryACK_ACK)
	case central.NodeInventoryACK_NACK:
		c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(), sensor.MsgToCompliance_NodeInventoryACK_NACK)
	}
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeInventoryHandlerImpl) run() (<-chan *message.ExpiringMessage, <-chan common.MessageToComplianceWithAddress) {
	toCentral := make(chan *message.ExpiringMessage)
	toCompliance := make(chan common.MessageToComplianceWithAddress)

	go c.nodeInventoryHandlingLoop(toCentral, toCompliance)

	return toCentral, toCompliance
}

func (c *nodeInventoryHandlerImpl) nodeInventoryHandlingLoop(toCentral chan *message.ExpiringMessage, toCompliance chan common.MessageToComplianceWithAddress) {
	defer c.stopper.Flow().ReportStopped()
	defer close(toCentral)
	defer close(toCompliance)
	for {
		select {
		case <-c.stopper.Flow().StopRequested():
			return
		case ackMsg, ok := <-c.acksFromCentral:
			if !ok {
				log.Debug("Channel for reading node-scanning-ACK messages (acksFromCentral) is closed")
			}
			log.Debugf("Forwarding node-scanning-ACK message from Central to Compliance: %v", ackMsg)
			toCompliance <- ackMsg
		case inventory, ok := <-c.inventories:
			if !ok {
				c.stopper.Flow().StopWithError(errInputChanClosed)
				return
			}
			if !c.centralReady.IsDone() {
				log.Warnf("Received NodeInventory but Central is not reachable. Requesting Compliance to resend NodeInventory later")
				c.sendAckToCompliance(toCompliance, inventory.GetNodeName(), sensor.MsgToCompliance_NodeInventoryACK_NACK)
				metrics.ObserveNodeInventoryAck(inventory.GetNodeName(),
					sensor.MsgToCompliance_NodeInventoryACK_NACK.String(),
					metrics.AckReasonCentralUnreachable, metrics.AckOriginSensor)
				continue
			}
			if inventory == nil {
				log.Warnf("Received nil node inventory: not sending to Central")
				break
			}
			if nodeID, err := c.nodeMatcher.GetNodeID(inventory.GetNodeName()); err != nil {
				log.Warnf("Node %q unknown to Sensor. Requesting Compliance to resend NodeInventory later", inventory.GetNodeName())
				c.sendAckToCompliance(toCompliance, inventory.GetNodeName(), sensor.MsgToCompliance_NodeInventoryACK_NACK)
				metrics.ObserveNodeInventoryAck(inventory.GetNodeName(), sensor.MsgToCompliance_NodeInventoryACK_NACK.String(),
					metrics.AckReasonNodeUnknown, metrics.AckOriginSensor)

			} else {
				inventory.NodeId = nodeID
				metrics.ObserveReceivedNodeInventory(inventory)
				log.Debugf("Mapping NodeInventory name '%s' to Node ID '%s'", inventory.GetNodeName(), nodeID)
				c.sendNodeInventory(toCentral, inventory)
			}
		}
	}
}

func (c *nodeInventoryHandlerImpl) sendAckToCompliance(complianceC chan<- common.MessageToComplianceWithAddress,
	nodeName string, action sensor.MsgToCompliance_NodeInventoryACK_Action) {
	complianceC <- common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_Ack{
				Ack: &sensor.MsgToCompliance_NodeInventoryACK{
					Action: action,
				},
			},
		},
		Hostname:  nodeName,
		Broadcast: nodeName == "",
	}
}

func (c *nodeInventoryHandlerImpl) sendNodeInventory(toC chan<- *message.ExpiringMessage, inventory *storage.NodeInventory) {
	if inventory == nil {
		return
	}
	select {
	case <-c.stopper.Flow().StopRequested():
	case toC <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     inventory.GetNodeId(),
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE, // There is no action required for NodeInventory as this is not a K8s resource
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: inventory,
				},
			},
		},
	}):
	}
}
