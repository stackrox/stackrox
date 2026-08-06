package compliance

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	errInventoryInputChanClosed = errors.New("channel receiving node inventories is closed")
	errIndexInputChanClosed     = errors.New("channel receiving node indexes is closed")
	errStartMoreThanOnce        = errors.New("unable to start the component more than once")
)

type nodeInventoryHandlerImpl struct {
	inventories  <-chan *storage.NodeInventory
	reportWraps  <-chan *index.IndexReportWrap
	toCentral    <-chan *message.ExpiringMessage
	centralReady concurrency.Signal
	// ch2Central is the same channel exposed read-only as toCentral; kept as a field so both
	// the legacy select loop (run()) and the PubSub event handlers can write to it. It is never
	// closed: PubSub callbacks can still be invoked after Stop() (the dispatcher has no
	// per-consumer unregistration), and writing to it must stay safe for the process lifetime.
	ch2Central       chan *message.ExpiringMessage
	toCompliance     chan common.MessageToComplianceWithAddress
	nodeMatcher      NodeIDMatcher
	nodeRHCOSMatcher NodeRHCOSMatcher
	// lock prevents the race condition between Start() [writer] and ResponsesC() [reader]
	lock             *sync.Mutex
	stopper          concurrency.Stopper
	pubSubDispatcher common.PubSubDispatcher
}

func (c *nodeInventoryHandlerImpl) Name() string {
	return "compliance.nodeInventoryHandlerImpl"
}

func (c *nodeInventoryHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

func (c *nodeInventoryHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SensorACKSupport}
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
	// Created here (before consumer registration) so handleNodeInventoryEvent/handleNodeIndexEvent
	// never observe a nil channel if a PubSub event is delivered before run() would otherwise create it.
	c.ch2Central = make(chan *message.ExpiringMessage)
	if features.SensorInternalPubSub.Enabled() && c.pubSubDispatcher != nil {
		if err := c.pubSubDispatcher.RegisterConsumerToLane(
			pubsub.NodeInventoryHandlerNodeInventoryConsumer,
			pubsub.NodeInventoryTopic,
			pubsub.NodeInventoryIntakeLane,
			c.handleNodeInventoryEvent,
		); err != nil {
			return errors.Wrap(err, "failed to register node inventory consumer")
		}
		if err := c.pubSubDispatcher.RegisterConsumerToLane(
			pubsub.NodeInventoryHandlerIndexReportConsumer,
			pubsub.IndexReportWrapTopic,
			pubsub.NodeInventoryIntakeLane,
			c.handleNodeIndexEvent,
		); err != nil {
			return errors.Wrap(err, "failed to register index report consumer")
		}
	}
	c.toCompliance = make(chan common.MessageToComplianceWithAddress)
	c.toCentral = c.run()
	return nil
}

func (c *nodeInventoryHandlerImpl) Stop() {
	if !c.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(c.stopper.Client().Stopped().Wait)
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

func (c *nodeInventoryHandlerImpl) Accepts(msg *central.MsgToSensor) bool {
	if msg.GetNodeInventoryAck() != nil {
		return true
	}
	if sensorAck := msg.GetSensorAck(); sensorAck != nil {
		switch sensorAck.GetMessageType() {
		case central.SensorACK_NODE_INVENTORY, central.SensorACK_NODE_INDEX_REPORT:
			return true
		}
	}
	return false
}

func (c *nodeInventoryHandlerImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	// Handle new SensorACK message (from Central 4.10+)
	if sensorAck := msg.GetSensorAck(); sensorAck != nil {
		return c.processSensorACK(sensorAck)
	}

	// Handle legacy NodeInventoryACK message (from Central 4.9 and earlier)
	if ackMsg := msg.GetNodeInventoryAck(); ackMsg != nil {
		return c.processNodeInventoryACK(ackMsg)
	}

	return nil
}

// processSensorACK handles the new generic SensorACK message from Central.
// Only node-related ACK/NACK messages (NODE_INVENTORY, NODE_INDEX_REPORT) are forwarded to Compliance.
// All other message types are ignored - they should be handled by their respective handlers.
func (c *nodeInventoryHandlerImpl) processSensorACK(sensorAck *central.SensorACK) error {
	log.Debugf("Received SensorACK message: type=%s, action=%s, resource_id=%s, reason=%s",
		sensorAck.GetMessageType(), sensorAck.GetAction(), sensorAck.GetResourceId(), sensorAck.GetReason())

	metrics.ObserveNodeScanningAck(sensorAck.GetResourceId(),
		sensorAck.GetAction().String(),
		sensorAck.GetMessageType().String(),
		metrics.AckOperationReceive,
		"", metrics.AckOriginSensor)

	// Only handle node-related message types - all others are handled by their respective handlers
	var messageType sensor.MsgToCompliance_ComplianceACK_MessageType
	switch sensorAck.GetMessageType() {
	case central.SensorACK_NODE_INVENTORY:
		messageType = sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY
	case central.SensorACK_NODE_INDEX_REPORT:
		messageType = sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT
	default:
		// Not a node-related message - ignore it (handled by other handlers like VM handler)
		log.Debugf("Ignoring SensorACK message type %s - not handled by node inventory handler", sensorAck.GetMessageType())
		return nil
	}

	// Map central.SensorACK action to sensor.ComplianceACK action
	var action sensor.MsgToCompliance_ComplianceACK_Action
	switch sensorAck.GetAction() {
	case central.SensorACK_ACK:
		action = sensor.MsgToCompliance_ComplianceACK_ACK
	case central.SensorACK_NACK:
		action = sensor.MsgToCompliance_ComplianceACK_NACK
	default:
		log.Debugf("Ignoring SensorACK message with unknown action %s: type=%s, resource_id=%s, reason=%s",
			sensorAck.GetAction(), sensorAck.GetMessageType(), sensorAck.GetResourceId(), sensorAck.GetReason())
		return nil
	}

	c.sendComplianceAck(
		sensorAck.GetResourceId(),
		action,
		messageType,
		sensorAck.GetReason(),
		metrics.AckReasonForwardingFromCentral,
	)
	return nil
}

// processNodeInventoryACK handles the legacy NodeInventoryACK message from Central 4.9 and earlier.
// It forwards the ACK/NACK to Compliance using the legacy NodeInventoryACK message type.
func (c *nodeInventoryHandlerImpl) processNodeInventoryACK(ackMsg *central.NodeInventoryACK) error {
	log.Debugf("Received legacy node-scanning-ACK message of type %s, action %s for node %s",
		ackMsg.GetMessageType(), ackMsg.GetAction(), ackMsg.GetNodeName())
	metrics.ObserveNodeScanningAck(ackMsg.GetNodeName(),
		ackMsg.GetAction().String(),
		ackMsg.GetMessageType().String(),
		metrics.AckOperationReceive,
		"", metrics.AckOriginSensor)

	var action sensor.MsgToCompliance_ComplianceACK_Action
	switch ackMsg.GetAction() {
	case central.NodeInventoryACK_ACK:
		action = sensor.MsgToCompliance_ComplianceACK_ACK
	case central.NodeInventoryACK_NACK:
		action = sensor.MsgToCompliance_ComplianceACK_NACK
	default:
		log.Debugf("Ignoring legacy NodeInventoryACK with unknown action %s", ackMsg.GetAction())
		return nil
	}

	var messageType sensor.MsgToCompliance_ComplianceACK_MessageType
	switch ackMsg.GetMessageType() {
	case central.NodeInventoryACK_NodeIndexer:
		messageType = sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT
	default:
		// If Central version is behind Sensor, MessageType can be unset: default to node inventory.
		messageType = sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY
	}

	c.sendComplianceAck(
		ackMsg.GetNodeName(),
		action,
		messageType,
		"",
		metrics.AckReasonForwardingFromCentral,
	)

	return nil
}

// run handles the messages from Compliance and forwards them to Central. It is one of possibly
// several writers into the toCentral channel (PubSub event handlers are the others); it never
// closes that channel since it cannot guarantee no PubSub handler will write after it returns.
func (c *nodeInventoryHandlerImpl) run() (toCentral <-chan *message.ExpiringMessage) {
	go func() {
		defer c.stopper.Flow().ReportStopped()
		log.Debugf("NodeInventory/NodeIndex handler is running")
		for {
			select {
			case <-c.stopper.Flow().StopRequested():
				return
			case inventory, ok := <-c.inventories:
				if !ok {
					c.stopper.Flow().StopWithError(errInventoryInputChanClosed)
					return
				}
				c.handleNodeInventory(inventory, c.ch2Central)
			case wrap, ok := <-c.reportWraps:
				if !ok {
					c.stopper.Flow().StopWithError(errIndexInputChanClosed)
					return
				}
				c.handleNodeIndex(wrap, c.ch2Central)
			}
		}
	}()
	return c.ch2Central
}

// handleNodeInventoryEvent adapts a PubSub NodeInventoryEvent to handleNodeInventory.
func (c *nodeInventoryHandlerImpl) handleNodeInventoryEvent(event pubsub.Event) error {
	inventoryEvent, ok := event.(*NodeInventoryEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", event)
	}
	c.handleNodeInventory(inventoryEvent.Inventory, c.ch2Central)
	return nil
}

// handleNodeIndexEvent adapts a PubSub IndexReportWrapEvent to handleNodeIndex.
func (c *nodeInventoryHandlerImpl) handleNodeIndexEvent(event pubsub.Event) error {
	wrapEvent, ok := event.(*IndexReportWrapEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", event)
	}
	c.handleNodeIndex(wrapEvent.Wrap, c.ch2Central)
	return nil
}

func (c *nodeInventoryHandlerImpl) handleNodeInventory(
	inventory *storage.NodeInventory,
	toCentral chan *message.ExpiringMessage,
) {
	log.Debugf("Handling NodeInventory...")
	if inventory == nil {
		log.Warn("Received nil node inventory: not sending to Central")
		metrics.ObserveNodeScan("nil", metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationReceive)
		return
	}
	metrics.ObserveNodeScan(inventory.GetNodeName(), metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationReceive)
	if !c.centralReady.IsDone() {
		log.Warn("Received NodeInventory but Central is not reachable. Requesting Compliance to resend NodeInventory later")
		c.sendComplianceAck(
			inventory.GetNodeName(),
			sensor.MsgToCompliance_ComplianceACK_NACK,
			sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
			string(metrics.AckReasonCentralUnreachable),
			metrics.AckReasonCentralUnreachable,
		)
		return
	}

	if nodeID, err := c.nodeMatcher.GetNodeID(inventory.GetNodeName()); err != nil {
		log.Warnf("Node %q unknown to Sensor. Requesting Compliance to resend NodeInventory later", inventory.GetNodeName())
		c.sendComplianceAck(
			inventory.GetNodeName(),
			sensor.MsgToCompliance_ComplianceACK_NACK,
			sensor.MsgToCompliance_ComplianceACK_NODE_INVENTORY,
			string(metrics.AckReasonNodeUnknown),
			metrics.AckReasonNodeUnknown,
		)

	} else {
		inventory.NodeId = nodeID
		log.Debugf("Mapping NodeInventory name '%s' to Node ID '%s'", inventory.GetNodeName(), nodeID)
		c.sendNodeInventory(toCentral, inventory)
	}
}

func (c *nodeInventoryHandlerImpl) handleNodeIndex(
	index *index.IndexReportWrap,
	toCentral chan *message.ExpiringMessage,
) {
	if index == nil || index.IndexReport == nil {
		log.Warn("Received nil index report: not sending to Central")
		metrics.ObserveNodeScan("nil", metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationReceive)
		return
	}
	metrics.ObserveNodeScan(index.NodeName, metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationReceive)
	if !c.centralReady.IsDone() {
		log.Warn("Received IndexReport but Central is not reachable. Requesting Compliance to resend later.")
		c.sendComplianceAck(
			index.NodeName,
			sensor.MsgToCompliance_ComplianceACK_NACK,
			sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
			string(metrics.AckReasonCentralUnreachable),
			metrics.AckReasonCentralUnreachable,
		)
		return
	}

	if nodeID, err := c.nodeMatcher.GetNodeID(index.NodeName); err != nil {
		log.Warnf("Received Index Report from Node %q that is unknown to Sensor. Requesting Compliance to resend later.", index.NodeName)
		c.sendComplianceAck(
			index.NodeName,
			sensor.MsgToCompliance_ComplianceACK_NACK,
			sensor.MsgToCompliance_ComplianceACK_NODE_INDEX_REPORT,
			string(metrics.AckReasonNodeUnknown),
			metrics.AckReasonNodeUnknown,
		)
	} else {
		index.NodeID = nodeID
		log.Debugf("Mapping IndexReport name '%s' to Node ID '%s'", index.NodeName, nodeID)
		c.sendNodeIndex(toCentral, index)
	}
}

// sendComplianceAck sends a ComplianceACK message to Compliance.
func (c *nodeInventoryHandlerImpl) sendComplianceAck(
	resourceID string,
	action sensor.MsgToCompliance_ComplianceACK_Action,
	messageType sensor.MsgToCompliance_ComplianceACK_MessageType,
	reason string,
	metricReason metrics.AckReason,
) {
	msg := common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_ComplianceAck{
				ComplianceAck: &sensor.MsgToCompliance_ComplianceACK{
					Action:      action,
					MessageType: messageType,
					ResourceId:  resourceID,
					Reason:      reason,
				},
			},
		},
		Hostname:  resourceID, // For node-based messages, resourceID is the node name
		Broadcast: resourceID == "",
	}

	if features.SensorInternalPubSub.Enabled() && c.pubSubDispatcher != nil {
		if err := c.pubSubDispatcher.Publish(&ComplianceAckEvent{Msg: msg}); err != nil {
			logging.GetRateLimitedLogger().ErrorL("compliance-ack-publish", "Failed to publish compliance ack event: %v", err)
			return
		}
	} else {
		select {
		case <-c.stopper.Flow().StopRequested():
			log.Debugf("Skipped sending ComplianceACK (stop requested): type=%s, action=%s, resource_id=%s, reason=%s",
				messageType, action, resourceID, reason)
			return
		case c.toCompliance <- msg:
		}
	}

	log.Debugf("Sent ComplianceACK to Compliance: type=%s, action=%s, resource_id=%s, reason=%s",
		messageType, action, resourceID, reason)

	// Record old metric for compatibility.
	metrics.ObserveNodeScanningAck(resourceID,
		action.String(),
		messageType.String(),
		metrics.AckOperationSend,
		metricReason,
		metrics.AckOriginSensor)
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
				// This can be changed to CREATE or UPDATE for Sensor 4.8 or when Central 4.6 is out of support.
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE,
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: inventory,
				},
			},
		},
	}):
		metrics.ObserveNodeScan(inventory.GetNodeName(), metrics.NodeScanTypeNodeInventory, metrics.NodeScanOperationSendToCentral)
	}
}

func (c *nodeInventoryHandlerImpl) sendNodeIndex(toC chan<- *message.ExpiringMessage, indexWrap *index.IndexReportWrap) {
	if indexWrap == nil || indexWrap.IndexReport == nil {
		log.Debugf("Empty IndexReport - not sending to Central")
		return
	}

	if hasRHCOSPackage(indexWrap.IndexReport) {
		log.Debugf("Node=%q has rhcos package from compliance", indexWrap.NodeName)
	} else {
		isRHCOS, ver, err := c.nodeRHCOSMatcher.GetRHCOSVersion(indexWrap.NodeName)
		if err != nil {
			log.Debugf("Unable to determine RHCOS version for node %q: %v", indexWrap.NodeName, err)
		} else if isRHCOS {
			log.Warnf("Node %q appears to be RHCOS (osImage version=%s) but compliance did not add rhcos package - RHCOS-level vulnerabilities will not be reported", indexWrap.NodeName, ver)
		}
	}

	select {
	case <-c.stopper.Flow().StopRequested():
	case toC <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: indexWrap.NodeID,
				// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
				// This can be changed to CREATE or UPDATE for Sensor 4.8 or when Central 4.6 is out of support.
				Action: central.ResourceAction_UNSET_ACTION_RESOURCE,
				Resource: &central.SensorEvent_IndexReport{
					IndexReport: indexWrap.IndexReport,
				},
			},
		},
	}):
		log.Debugf("Sent IndexReport to Central")
		metrics.ObserveNodeScan(indexWrap.NodeName, metrics.NodeScanTypeNodeIndex, metrics.NodeScanOperationSendToCentral)
	}
}

func hasRHCOSPackage(report *v4.IndexReport) bool {
	for _, p := range report.GetContents().GetPackages() {
		if p.GetName() == "rhcos" {
			return true
		}
	}
	return false
}
