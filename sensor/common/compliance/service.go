package compliance

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/sensor/common/orchestrator"
)

// Service is an interface to receiving ComplianceReturns from launched daemons.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.ComplianceServiceServer

	RunScrape(msg *sensor.MsgToCompliance) int
	Output() chan *compliance.ComplianceReturn
}

// NewService returns the ComplianceServiceServer API for Sensor to use, outputs any received ComplianceReturns
// to the input channel.
func NewService(orchestrator orchestrator.Orchestrator) Service {
	return &serviceImpl{
		output:            make(chan *compliance.ComplianceReturn),
		connectionManager: newConnectionManager(),
		orchestrator:      orchestrator,
	}
}
