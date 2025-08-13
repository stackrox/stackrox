package index

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
)

// Service provides an API to upsert virtual machine index reports to Central.
type Service interface {
	grpcPkg.APIService
	sensor.VirtualMachineIndexReportServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// NewService returns the VirtualMachineIndexReportServiceServer API for Sensor to use.
func NewService(handler Handler) Service {
	return &serviceImpl{handler: handler}
}
