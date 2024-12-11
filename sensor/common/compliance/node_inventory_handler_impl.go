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
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	errInputChanClosed   = errors.New("channel receiving node inventories is closed")
	errStartMoreThanOnce = errors.New("unable to start the component more than once")
)

type nodeInventoryHandlerImpl struct {
	inventories  <-chan *storage.NodeInventory
	reportWraps  <-chan *index.IndexReportWrap
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
	log.Debugf("Received node-scanning-ACK message of type %s, action %s for node %s",
		ackMsg.GetMessageType(), ackMsg.GetAction(), ackMsg.GetNodeName())
	switch ackMsg.GetAction() {
	case central.NodeInventoryACK_ACK:
		switch ackMsg.GetMessageType() {
		case central.NodeInventoryACK_NodeIndexer:
			c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_ACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer, metrics.AckReasonForwardingFromCentral)
		default:
			// If Central version is behind Sensor, then MessageType field will be unset - then default to NodeInventory.
			c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_ACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonForwardingFromCentral)
		}
	case central.NodeInventoryACK_NACK:
		switch ackMsg.GetMessageType() {
		case central.NodeInventoryACK_NodeIndexer:
			c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_NACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer, metrics.AckReasonForwardingFromCentral)
		default:
			// If Central version is behind Sensor, then MessageType field will be unset - then default to NodeInventory.
			c.sendAckToCompliance(c.acksFromCentral, ackMsg.GetNodeName(),
				sensor.MsgToCompliance_NodeInventoryACK_NACK,
				sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonForwardingFromCentral)
		}
	}
	return nil
}

// run handles the messages from Compliance and forwards them to Central
// This is the only goroutine that writes into the toCentral channel, thus it is responsible for creating and closing that chan
func (c *nodeInventoryHandlerImpl) run() (<-chan *message.ExpiringMessage, <-chan common.MessageToComplianceWithAddress) {
	toCentral := make(chan *message.ExpiringMessage)
	toCompliance := make(chan common.MessageToComplianceWithAddress)

	go c.nodeScanHandlingLoop(toCentral, toCompliance)

	return toCentral, toCompliance
}

func (c *nodeInventoryHandlerImpl) nodeScanHandlingLoop(toCentral chan *message.ExpiringMessage, toCompliance chan common.MessageToComplianceWithAddress) {
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
			if ackMsg.Msg != nil {
				log.Debugf("Forwarding node-scanning-ACK message of type %s, action %s for node %s",
					ackMsg.Msg.GetAck().GetMessageType(), ackMsg.Msg.GetAck().GetAction(), ackMsg.Hostname)
			}
			toCompliance <- ackMsg
		case inventory, ok := <-c.inventories:
			if !ok {
				c.stopper.Flow().StopWithError(errInputChanClosed)
				return
			}
			if !c.centralReady.IsDone() {
				log.Warn("Received NodeInventory but Central is not reachable. Requesting Compliance to resend NodeInventory later")
				c.sendAckToCompliance(toCompliance, inventory.GetNodeName(),
					sensor.MsgToCompliance_NodeInventoryACK_NACK,
					sensor.MsgToCompliance_NodeInventoryACK_NodeInventory, metrics.AckReasonCentralUnreachable)
				continue
			}
			if inventory == nil {
				log.Warn("Received nil node inventory: not sending to Central")
				continue
			}
			if nodeID, err := c.nodeMatcher.GetNodeID(inventory.GetNodeName()); err != nil {
				log.Warnf("Node %q unknown to Sensor. Requesting Compliance to resend NodeInventory later", inventory.GetNodeName())
				c.sendAckToCompliance(toCompliance, inventory.GetNodeName(),
					sensor.MsgToCompliance_NodeInventoryACK_NACK,
					sensor.MsgToCompliance_NodeInventoryACK_NodeInventory,
					metrics.AckReasonNodeUnknown)

			} else {
				inventory.NodeId = nodeID
				metrics.ObserveReceivedNodeInventory(inventory)
				log.Debugf("Mapping NodeInventory name '%s' to Node ID '%s'", inventory.GetNodeName(), nodeID)
				c.sendNodeInventory(toCentral, inventory)
			}
		case wrap, ok := <-c.reportWraps:
			if !ok {
				log.Warn("Report wraps channel is closed")
				c.stopper.Flow().StopWithError(errInputChanClosed)
				return
			}
			if !c.centralReady.IsDone() {
				log.Warn("Received Index Report but Central is not reachable. Requesting Compliance to resend later.")
				c.sendAckToCompliance(toCompliance, wrap.NodeName,
					sensor.MsgToCompliance_NodeInventoryACK_NACK,
					sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer,
					metrics.AckReasonCentralUnreachable)
				continue
			}
			if wrap == nil || wrap.IndexReport == nil {
				log.Warn("Received nil index report: not sending to Central")
				continue
			}
			if nodeID, err := c.nodeMatcher.GetNodeID(wrap.NodeName); err != nil {
				log.Warnf("Received Index Report from Node %q that is unknown to Sensor. Requesting Compliance to resend later.", wrap.NodeName)
				c.sendAckToCompliance(toCompliance, wrap.NodeName,
					sensor.MsgToCompliance_NodeInventoryACK_NACK,
					sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer,
					metrics.AckReasonNodeUnknown)
			} else {
				wrap.NodeID = nodeID
				metrics.ObserveReceivedNodeIndex(wrap.NodeName)
				log.Debugf("Mapping Node Index name '%s' to Node ID '%s'", wrap.NodeName, nodeID)
				c.sendNodeIndex(toCentral, wrap)
			}
		}
	}
}

func (c *nodeInventoryHandlerImpl) sendAckToCompliance(
	complianceC chan<- common.MessageToComplianceWithAddress,
	nodeName string,
	action sensor.MsgToCompliance_NodeInventoryACK_Action,
	messageType sensor.MsgToCompliance_NodeInventoryACK_MessageType,
	reason metrics.AckReason,
) {
	complianceC <- common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_Ack{
				Ack: &sensor.MsgToCompliance_NodeInventoryACK{
					Action:      action,
					MessageType: messageType,
				},
			},
		},
		Hostname:  nodeName,
		Broadcast: nodeName == "",
	}
	metrics.ObserveNodeScanningAck(nodeName,
		action.String(),
		messageType.String(),
		reason, metrics.AckOriginSensor)
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
				Id: inventory.GetNodeId(),
				// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
				// This can be changed to CREATE or UPDATE when 4.6 is out of support.
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE,
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: inventory,
				},
			},
		},
	}):
	}
}

func (c *nodeInventoryHandlerImpl) sendNodeIndex(toC chan<- *message.ExpiringMessage, indexWrap *index.IndexReportWrap) {
	if indexWrap == nil || indexWrap.IndexReport == nil {
		log.Debugf("Empty index report - not sending to Central")
		return
	}
	defer log.Debugf("Finished sending index report to Central")
	select {
	case <-c.stopper.Flow().StopRequested():
	case toC <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: indexWrap.NodeID,
				// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
				// This can be changed to CREATE or UPDATE when 4.6 is out of support.
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE,
				Resource: &central.SensorEvent_IndexReport{
					IndexReport: indexWrap.IndexReport,
				},
			},
		},
	}):
	}
}
