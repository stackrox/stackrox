package compliance

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/orchestrator"
)

//go:generate mockgen-wrapper

// Service is the sensor-side compliance service for node inventory, audit logs, and index reports.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.ComplianceServiceServer
	common.SensorComponent

	AuditEvents() chan *sensor.AuditEvents
	NodeInventories() <-chan *storage.NodeInventory
	IndexReportWraps() <-chan *index.IndexReportWrap
}

// NewService returns the ComplianceServiceServer API for Sensor to use.
func NewService(orchestrator orchestrator.Orchestrator, auditEventsInput chan *sensor.AuditEvents, auditLogCollectionManager AuditLogCollectionManager, complianceC <-chan common.MessageToComplianceWithAddress, pubSubDispatcher common.PubSubDispatcher) Service {
	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)
	return &serviceImpl{
		nodeInventories:           make(chan *storage.NodeInventory),
		indexReportWraps:          make(chan *index.IndexReportWrap),
		complianceC:               complianceC,
		orchestrator:              orchestrator,
		auditEvents:               auditEventsInput,
		auditLogCollectionManager: auditLogCollectionManager,
		pubSubDispatcher:          pubSubDispatcher,
		connectionManager:         newConnectionManager(),
		offlineMode:               offlineMode,
		stopper:                   set.NewSet[concurrency.Stopper](),
	}
}
