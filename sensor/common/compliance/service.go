package compliance

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/compliance"
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

// Service is an interface to receiving ComplianceReturns from launched daemons.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.ComplianceServiceServer
	common.SensorComponent

	RunScrape(msg *sensor.MsgToCompliance) int

	Output() chan *compliance.ComplianceReturn
	AuditEvents() chan *sensor.AuditEvents
	NodeInventories() <-chan *storage.NodeInventory
	IndexReports() <-chan *index.Report
}

// NewService returns the ComplianceServiceServer API for Sensor to use, outputs any received ComplianceReturns
// to the input channel.
func NewService(orchestrator orchestrator.Orchestrator, auditEventsInput chan *sensor.AuditEvents, auditLogCollectionManager AuditLogCollectionManager, complianceC <-chan common.MessageToComplianceWithAddress) Service {
	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)
	return &serviceImpl{
		output:                    make(chan *compliance.ComplianceReturn),
		nodeInventories:           make(chan *storage.NodeInventory),
		indexReports:              make(chan *index.Report),
		complianceC:               complianceC,
		orchestrator:              orchestrator,
		auditEvents:               auditEventsInput,
		auditLogCollectionManager: auditLogCollectionManager,
		connectionManager:         newConnectionManager(),
		offlineMode:               offlineMode,
		stopper:                   set.NewSet[concurrency.Stopper](),
	}
}
