package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterid"
)

// AuditLogCollectionManager manages the lifecycle of audit log collection within the cluster
type AuditLogCollectionManager struct {
	enabled         bool
	fileStates      map[string]*storage.AuditLogFileState
	clusterIDGetter func() string

	eligibleComplianceNodes map[string]sensor.ComplianceService_CommunicateServer

	lock           sync.RWMutex
	connectionLock sync.RWMutex
}

// NewAuditLogCollectionManager returns the API for sensor to use to add eligible nodes, enable/disable/restart collection
func NewAuditLogCollectionManager() *AuditLogCollectionManager {
	return &AuditLogCollectionManager{
		eligibleComplianceNodes: make(map[string]sensor.ComplianceService_CommunicateServer),
		// Need to use a getter instead of directly calling clusterid.Get because it may block if the communication with central is not yet finished
		// Putting it as a getter so it can be overridden in tests
		clusterIDGetter: clusterid.Get,
	}
}

// AddEligibleComplianceNode adds the specified node and it's connection to the list of nodes whose audit log collection lifecycle will be managed
// If the feature is enabled, then the node will automatically be sent a message to start collection upon a successful add
func (a *AuditLogCollectionManager) AddEligibleComplianceNode(node string, connection sensor.ComplianceService_CommunicateServer) {
	a.connectionLock.Lock()
	a.eligibleComplianceNodes[node] = connection
	a.connectionLock.Unlock()

	a.lock.RLock() // locked because we will need to read state when enabling collection
	defer a.lock.RUnlock()
	if a.enabled {
		a.startCollectionOnNode(node, connection)
	}
}

// RemoveEligibleComplianceNode removes the specified node and it's connection from the list of nodes whose audit log collection lifecycle will be managed
func (a *AuditLogCollectionManager) RemoveEligibleComplianceNode(node string) {
	log.Infof("Removing node `%s` as an eligible compliance node for audit log collection", node)
	a.connectionLock.Lock()
	defer a.connectionLock.Unlock()

	delete(a.eligibleComplianceNodes, node)

	// Not sending a stop message because it is likely the connection has closed by this point
}

func (a *AuditLogCollectionManager) forEachNode(fn func(node string, server sensor.ComplianceService_CommunicateServer)) {
	a.connectionLock.RLock()
	defer a.connectionLock.RUnlock()

	for node, server := range a.eligibleComplianceNodes {
		fn(node, server)
	}
}

// EnableCollection enables audit log collection on all the nodes who are eligible
func (a *AuditLogCollectionManager) EnableCollection() {
	if !features.K8sAuditLogDetection.Enabled() {
		log.Error("Request to enable audit log collection when feature is disabled!")
		return
	}
	a.lock.Lock()
	a.enabled = true
	a.lock.Unlock()

	a.lock.RLock() // locked because we will need to read state when enabling collection
	defer a.lock.RUnlock()
	a.startAuditLogCollectionOnAllNodes()
}

func (a *AuditLogCollectionManager) startAuditLogCollectionOnAllNodes() {
	a.forEachNode(func(node string, server sensor.ComplianceService_CommunicateServer) {
		a.startCollectionOnNode(node, server)
	})
}

func (a *AuditLogCollectionManager) startCollectionOnNode(node string, server sensor.ComplianceService_CommunicateServer) {
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
		log.Errorf("error sending audit log collection start request to node %q: %v", node, err)
	}
}

// DisableCollection disables audit log collection on all the nodes who are eligible
func (a *AuditLogCollectionManager) DisableCollection() {
	if !features.K8sAuditLogDetection.Enabled() {
		log.Error("Request to enable audit log collection when feature is disabled!")
		return
	}
	a.lock.Lock()
	a.enabled = false
	a.lock.Unlock()

	a.stopAuditLogCollectionOnAllNodes()
}

func (a *AuditLogCollectionManager) stopAuditLogCollectionOnAllNodes() {
	if !features.K8sAuditLogDetection.Enabled() {
		log.Error("Request to disable audit log collection when feature is disabled!")
		return
	}

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

// UpdateAuditLogFileState updates the location at which each node collects audit logs
// If the feature is already enabled and there are eligible nodes, then this will restart collection on those nodes from this state
func (a *AuditLogCollectionManager) UpdateAuditLogFileState(fileStates map[string]*storage.AuditLogFileState) {
	if !features.K8sAuditLogDetection.Enabled() {
		log.Error("Request to enable audit log collection when feature is disabled!")
		return
	}
	a.lock.Lock()
	a.fileStates = fileStates
	a.lock.Unlock()

	a.lock.RLock() // locked because we will need to read state when enabling collection
	defer a.lock.RUnlock()
	if a.enabled {
		// No point in sending start if it's not even enabled
		a.startAuditLogCollectionOnAllNodes()
	}
}
