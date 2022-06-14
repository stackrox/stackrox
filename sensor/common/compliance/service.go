package compliance

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/sensor/common/orchestrator"
)

// Service is an interface to receiving ComplianceReturns from launched daemons.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.ComplianceServiceServer

	RunScrape(msg *sensor.MsgToCompliance) int

	Output() chan *compliance.ComplianceReturn
	AuditEvents() chan *sensor.AuditEvents
}

// NewService returns the ComplianceServiceServer API for Sensor to use, outputs any received ComplianceReturns
// to the input channel.
func NewService(orchestrator orchestrator.Orchestrator, auditEventsInput chan *sensor.AuditEvents, auditLogCollectionManager AuditLogCollectionManager) Service {
	return &serviceImpl{
		output:                    make(chan *compliance.ComplianceReturn),
		connectionManager:         newConnectionManager(),
		orchestrator:              orchestrator,
		auditEvents:               auditEventsInput,
		auditLogCollectionManager: auditLogCollectionManager,
	}
}
