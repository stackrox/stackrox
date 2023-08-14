package compliance

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/orchestrator"
)

//go:generate mockgen-wrapper Service

// Service is an interface to receiving ComplianceReturns from launched daemons.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.ComplianceServiceServer

	RunScrape(msg *sensor.MsgToCompliance) int

	Output() chan *compliance.ComplianceReturn
	AuditEvents() chan *sensor.AuditEvents
	NodeInventories() <-chan *storage.NodeInventory
}

// NewService returns the ComplianceServiceServer API for Sensor to use, outputs any received ComplianceReturns
// to the input channel.
func NewService(orchestrator orchestrator.Orchestrator, auditEventsInput chan *sensor.AuditEvents, auditLogCollectionManager AuditLogCollectionManager, complianceC <-chan common.MessageToComplianceWithAddress) Service {
	return &serviceImpl{
		output:                    make(chan *compliance.ComplianceReturn),
		nodeInventories:           make(chan *storage.NodeInventory),
		complianceC:               complianceC,
		orchestrator:              orchestrator,
		auditEvents:               auditEventsInput,
		auditLogCollectionManager: auditLogCollectionManager,
		connectionManager:         newConnectionManager(),
	}
}
