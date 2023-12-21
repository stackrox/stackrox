package compliance

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

const (
	defaultInterval = 1 * time.Minute
)

// auditLogCollectionManagerImpl manages the lifecycle of audit log collection within the cluster
type auditLogCollectionManagerImpl struct {
	clusterIDGetter func() string

	enabled                         concurrency.Flag
	receivedInitialStateFromCentral concurrency.Flag
	fileStates                      map[string]*storage.AuditLogFileState
	eligibleComplianceNodes         map[string]sensor.ComplianceService_CommunicateServer

	auditEventMsgs   chan *sensor.MsgFromCompliance
	fileStateUpdates chan *message.ExpiringMessage

	stopSig        concurrency.Signal
	forceUpdateSig concurrency.Signal
	centralReady   concurrency.Signal

	updateInterval time.Duration

	fileStateLock  sync.RWMutex
	connectionLock sync.RWMutex
}

func (a *auditLogCollectionManagerImpl) Start() error {
	go a.runStateSaver()
	go a.runUpdater()
	return nil
}

func (a *auditLogCollectionManagerImpl) Stop(_ error) {
	a.stopSig.Signal()
}

func (a *auditLogCollectionManagerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		a.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		a.centralReady.Reset()
	}
}

func (a *auditLogCollectionManagerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.AuditLogEventsCap}
}

func (a *auditLogCollectionManagerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	// This component doesn't actually process or handle any messages sent to Sensor. It uses the sensor component
	// so that the lifecycle (start, stop) can be handled when Sensor starts up. The actual messages from central to
	// enable/disable audit log collection is handled as part of the dynamic config in config.Handler which then calls
	// the specific APIs in this manager.
	return nil
}

func (a *auditLogCollectionManagerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return a.fileStateUpdates
}

// ForceUpdate immediately updates Central with the latest file state
func (a *auditLogCollectionManagerImpl) ForceUpdate() {
	// If the signal is already triggered then an update will happen soon (or is in process)
	// It will be reset once the update finishes
	a.forceUpdateSig.Signal()
}

func (a *auditLogCollectionManagerImpl) runStateSaver() {
	for {
		select {
		case <-a.stopSig.Done():
			return
		case msg := <-a.auditEventMsgs:
			node := msg.GetNode()
			if events := msg.GetAuditEvents(); len(events.GetEvents()) > 0 {
				// Given how audit logs are always in chronological order, and given how compliance is parsing it in said order,
				// we can make an assumption that the earliest event in this message is still later than the state before
				// But we won't check it, in case there is a corner case where the time is out of order.
				latestTime := events.GetEvents()[0].Timestamp
				latestID := events.GetEvents()[0].GetId()
				for _, e := range events.GetEvents()[1:] {
					if protoutils.After(e.GetTimestamp(), latestTime) {
						latestTime = e.GetTimestamp()
						latestID = e.GetId()
					}
				}
				a.updateFileState(node, latestTime, latestID)
			}
		}
	}
}

func (a *auditLogCollectionManagerImpl) updateFileState(node string, latestTime *types.Timestamp, latestID string) {
	a.fileStateLock.Lock()
	defer a.fileStateLock.Unlock()

	a.fileStates[node] = &storage.AuditLogFileState{
		CollectLogsSince: latestTime,
		LastAuditId:      latestID,
	}
}

func (a *auditLogCollectionManagerImpl) runUpdater() {
	ticker := time.NewTicker(a.updateInterval)
	defer ticker.Stop()

	for !a.stopSig.IsDone() {
		select {
		case <-a.stopSig.Done():
			return
		case <-a.forceUpdateSig.Done():
			a.sendUpdate()
			a.forceUpdateSig.Reset()
		case <-ticker.C:
			a.sendUpdate()
		}
	}
}

func (a *auditLogCollectionManagerImpl) sendUpdate() {
	fileStates := a.getLatestFileStates()

	if a.shouldSendUpdateToCentral(fileStates) {
		select {
		case a.fileStateUpdates <- a.getCentralUpdateMsg(fileStates):
		case <-a.stopSig.Done():
		}
	}
}

func (a *auditLogCollectionManagerImpl) shouldSendUpdateToCentral(fileStates map[string]*storage.AuditLogFileState) bool {
	// No point in updating if the central communication hasn't started, isn't available, or there are no states
	return a.receivedInitialStateFromCentral.Get() && a.centralReady.IsDone() && len(fileStates) > 0
}

// getLatestFileStates returns a copy of the latest state of audit log collection at each compliance node
func (a *auditLogCollectionManagerImpl) getLatestFileStates() map[string]*storage.AuditLogFileState {
	a.fileStateLock.RLock()
	defer a.fileStateLock.RUnlock()

	// Clone the map before returning because it may get changed before the caller has a chance to use it.
	nodeStates := make(map[string]*storage.AuditLogFileState, len(a.fileStates))
	for k, v := range a.fileStates {
		nodeStates[k] = v // no need to clone this because when the map is updated a new storage.AuditLogFileState is always created (see updateFileState)
	}
	return nodeStates
}

func (a *auditLogCollectionManagerImpl) getCentralUpdateMsg(fileStates map[string]*storage.AuditLogFileState) *message.ExpiringMessage {
	return message.New(&central.MsgFromSensor{
		HashKey:   a.clusterIDGetter(),
		DedupeKey: a.clusterIDGetter(),
		Msg: &central.MsgFromSensor_AuditLogStatusInfo{
			AuditLogStatusInfo: &central.AuditLogStatusInfo{
				NodeAuditLogFileStates: fileStates,
			},
		},
	})
}

// AddEligibleComplianceNode adds the specified node and it's connection to the list of nodes whose audit log collection lifecycle will be managed
// If the feature is enabled, then the node will automatically be sent a message to start collection upon a successful add
func (a *auditLogCollectionManagerImpl) AddEligibleComplianceNode(node string, connection sensor.ComplianceService_CommunicateServer) {
	log.Infof("Adding node `%s` as an eligible compliance node for audit log collection", node)
	a.connectionLock.Lock()
	a.eligibleComplianceNodes[node] = connection
	a.connectionLock.Unlock()

	if a.enabled.Get() {
		a.fileStateLock.RLock() // Will read the state when sending start message.
		defer a.fileStateLock.RUnlock()
		a.startCollectionOnNodeNoFileStateLock(node, connection)
	}
}

// RemoveEligibleComplianceNode removes the specified node and it's connection from the list of nodes whose audit log collection lifecycle will be managed
func (a *auditLogCollectionManagerImpl) RemoveEligibleComplianceNode(node string) {
	log.Infof("Removing node `%s` as an eligible compliance node for audit log collection", node)
	a.connectionLock.Lock()
	defer a.connectionLock.Unlock()

	delete(a.eligibleComplianceNodes, node)

	// Not sending a stop message because it is likely the connection has closed by this point
}

func (a *auditLogCollectionManagerImpl) forEachNode(fn func(node string, server sensor.ComplianceService_CommunicateServer)) {
	a.connectionLock.RLock()
	defer a.connectionLock.RUnlock()

	for node, server := range a.eligibleComplianceNodes {
		fn(node, server)
	}
}

// EnableCollection enables audit log collection on all nodes that are eligible
func (a *auditLogCollectionManagerImpl) EnableCollection() {
	if wasEnabled := a.enabled.TestAndSet(true); !wasEnabled {
		a.startAuditLogCollectionOnAllNodes()
	}
}

func (a *auditLogCollectionManagerImpl) startAuditLogCollectionOnAllNodes() {
	// locked because we will need to read states when enabling collection
	a.fileStateLock.RLock()
	defer a.fileStateLock.RUnlock()
	a.forEachNode(a.startCollectionOnNodeNoFileStateLock)
}

// the fileStateLock lock must be acquired (in read mode) before calling this
func (a *auditLogCollectionManagerImpl) startCollectionOnNodeNoFileStateLock(node string, server sensor.ComplianceService_CommunicateServer) {
	log.Infof("Sending start audit log collection message to node %s", node)
	msg := &sensor.MsgToCompliance{
		Msg: &sensor.MsgToCompliance_AuditLogCollectionRequest_{
			AuditLogCollectionRequest: &sensor.MsgToCompliance_AuditLogCollectionRequest{
				Req: &sensor.MsgToCompliance_AuditLogCollectionRequest_StartReq{
					StartReq: &sensor.MsgToCompliance_AuditLogCollectionRequest_StartRequest{
						ClusterId: a.clusterIDGetter(),
					},
				},
			},
		},
	}
	if state := a.fileStates[node]; state != nil {
		log.Infof("Start message to node %s contains state %s", node, protoutils.NewWrapper(state))
		msg.GetAuditLogCollectionRequest().GetStartReq().CollectStartState = state
	}

	if err := server.Send(msg); err != nil {
		// TODO: Update health status if this fails. For now just log and move on
		log.Infof("error sending audit log collection start request to node %q: %v", node, err)
	}
}

// DisableCollection disables audit log collection on all nodes that are eligible
func (a *auditLogCollectionManagerImpl) DisableCollection() {
	if wasEnabled := a.enabled.TestAndSet(false); wasEnabled {
		a.stopAuditLogCollectionOnAllNodes()
	}
}

func (a *auditLogCollectionManagerImpl) stopAuditLogCollectionOnAllNodes() {
	a.forEachNode(func(node string, server sensor.ComplianceService_CommunicateServer) {
		log.Infof("Sending stop audit log collection message to node %s", node)
		msg := &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_AuditLogCollectionRequest_{
				AuditLogCollectionRequest: &sensor.MsgToCompliance_AuditLogCollectionRequest{
					Req: &sensor.MsgToCompliance_AuditLogCollectionRequest_StopReq{
						StopReq: &sensor.MsgToCompliance_AuditLogCollectionRequest_StopRequest{},
					},
				},
			},
		}

		if err := server.Send(msg); err != nil {
			// TODO: Update health status if this fails. For now just log and move on
			log.Errorf("error sending audit log collection stop request to node %q: %v", node, err)
			return
		}
	})
}

// SetAuditLogFileStateFromCentral sets the location at which each node should collect audit logs as sent by Central
// If the feature is already enabled and there are eligible nodes, then this will restart collection on those nodes from this state
func (a *auditLogCollectionManagerImpl) SetAuditLogFileStateFromCentral(fileStates map[string]*storage.AuditLogFileState) {
	a.receivedInitialStateFromCentral.Set(true)

	concurrency.WithLock(&a.fileStateLock, func() {
		a.fileStates = fileStates

		// Ensure that the map is empty not nil if there is no saved state. The rest of the manager depends on it being _not nil_
		if a.fileStates == nil {
			a.fileStates = make(map[string]*storage.AuditLogFileState)
		}
	})

	if a.enabled.Get() {
		a.startAuditLogCollectionOnAllNodes()
	}
}

// AuditMessagesChan returns a send-only channel that can be used to notify the manager of the latest received audit log message from a compliance node. It used to maintain the latest file states
func (a *auditLogCollectionManagerImpl) AuditMessagesChan() chan<- *sensor.MsgFromCompliance {
	return a.auditEventMsgs
}
