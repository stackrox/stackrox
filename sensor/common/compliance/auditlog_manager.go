package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/message"
)

//go:generate mockgen-wrapper

// AuditLogCollectionManager manages all aspects of the audit log collection states. Given the stream of audit events via the AuditMessages channel, it saves, keeps track and updates Central
// of the latest read audit log event per compliance node. It also provides an API for sensor to use to use to add eligible nodes, enable/disable/restart collection.
type AuditLogCollectionManager interface {
	// AddEligibleComplianceNode adds the specified node and its connection to the list of nodes whose audit log collection lifecycle will be managed
	// If the feature is enabled, then the node will automatically be sent a message to start collection upon a successful add
	AddEligibleComplianceNode(node string, connection sensor.ComplianceService_CommunicateServer)

	// RemoveEligibleComplianceNode removes the specified node and its connection from the list of nodes whose audit log collection lifecycle will be managed
	RemoveEligibleComplianceNode(node string)

	// EnableCollection enables audit log collection on all the nodes who are eligible
	EnableCollection()

	// DisableCollection disables audit log collection on all the nodes who are eligible
	DisableCollection()

	// SetAuditLogFileStateFromCentral sets the location at which each node should collect audit logs as sent by Central
	// If the feature is already enabled and there are eligible nodes, then this will restart collection on those nodes from this state
	SetAuditLogFileStateFromCentral(fileStates map[string]*storage.AuditLogFileState)

	// AuditMessagesChan returns a send-only channel that can be used to notify the manager of the latest received audit log message from a compliance node. It used to maintain the latest file states
	AuditMessagesChan() chan<- *sensor.MsgFromCompliance

	// ForceUpdate immediately updates Central with the latest file state
	ForceUpdate()

	common.SensorComponent
}

// NewAuditLogCollectionManager creates a new instance of AuditLogCollectionManager, which provides an API to manage the lifecycle of audit log collection within the cluster
func NewAuditLogCollectionManager() AuditLogCollectionManager {
	return &auditLogCollectionManagerImpl{
		// Need to use a getter instead of directly calling clusterid.Get because it may block if the communication with central is not yet finished
		// Putting it as a getter so it can be overridden in tests
		clusterIDGetter:         clusterid.Get,
		eligibleComplianceNodes: make(map[string]sensor.ComplianceService_CommunicateServer),
		fileStates:              make(map[string]*storage.AuditLogFileState),
		auditEventMsgs:          make(chan *sensor.MsgFromCompliance),
		fileStateUpdates:        make(chan *message.ExpiringMessage),
		stopSig:                 concurrency.NewSignal(),
		forceUpdateSig:          concurrency.NewSignal(),
		centralReady:            concurrency.NewSignal(),
		updateInterval:          defaultInterval,
	}
}
